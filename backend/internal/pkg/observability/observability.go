// Package observability exports content-free request telemetry to a local Collector.
package observability

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var tracer = noop.NewTracerProvider().Tracer("sub2api")
var log = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.Time(slog.TimeKey, a.Value.Time().UTC())
	}
	return a
}}))
var resourceFields = []any{"service.name", "sub2api", "service.namespace", "truerouter"}

// Setup is optional and fails open. No arbitrary environment resource detector runs.
func Setup(version string) func() {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" || (os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "") {
		return func() {}
	}
	instance := os.Getenv("TRUEROUTER_INSTANCE_ID")
	if instance == "" {
		instance = rand.Text()
	}
	environment := os.Getenv("TRUEROUTER_ENVIRONMENT")
	if environment == "" {
		environment = "unspecified"
	}
	resourceFields = append(resourceFields, "service.version", version, "service.instance.id", instance, "deployment.environment.name", environment)
	exporter, err := otlptracehttp.New(context.Background(), otlptracehttp.WithTimeout(3*time.Second))
	if err != nil {
		log.Warn("observability_setup_failed")
		return func() {}
	}
	res := resource.NewSchemaless(attribute.String("service.name", "sub2api"), attribute.String("service.namespace", "truerouter"), attribute.String("service.version", version), attribute.String("service.instance.id", instance), attribute.String("deployment.environment.name", environment))
	provider := sdk.NewTracerProvider(sdk.WithResource(res), sdk.WithSampler(sdk.ParentBased(sdk.AlwaysSample())), sdk.WithBatcher(exporter, sdk.WithMaxQueueSize(2048), sdk.WithMaxExportBatchSize(512), sdk.WithBatchTimeout(time.Second), sdk.WithExportTimeout(3*time.Second)))
	tracer = provider.Tracer("sub2api")
	log.Info("truerouter.observability", append(append([]any{}, resourceFields...), "event", "service_started")...)
	return func() {
		log.Info("truerouter.observability", append(append([]any{}, resourceFields...), "event", "service_stopping")...)
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		_ = provider.Shutdown(ctx)
	}
}

type requestStateKey struct{}

func RequestUnknown(ctx context.Context) bool {
	state, _ := ctx.Value(requestStateKey{}).(*atomic.Bool)
	return state != nil && state.Load()
}
func markUnknown(ctx context.Context) {
	if state, ok := ctx.Value(requestStateKey{}).(*atomic.Bool); ok {
		state.Store(true)
	}
}
func StartRequest(ctx context.Context) (context.Context, trace.Span) {
	ctx = context.WithValue(ctx, requestStateKey{}, &atomic.Bool{})
	return tracer.Start(ctx, "sub2api.request", trace.WithNewRoot(), trace.WithSpanKind(trace.SpanKindServer))
}

// Event only accepts caller-owned bounded metadata. Never pass payloads or raw errors.
func Event(ctx context.Context, event string, fields ...any) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return
	}
	args := append([]any{}, resourceFields...)
	args = append(args, "event", event, "trace_id", sc.TraceID().String(), "span_id", sc.SpanID().String())
	level := slog.LevelInfo
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] == "outcome" && (fields[i+1] == "failure" || fields[i+1] == "unknown") {
			level = slog.LevelError
		}
	}
	log.Log(ctx, level, "truerouter.observability", append(args, fields...)...)
}

// StreamState observes bounded protocol lines without retaining response content.
type StreamState struct {
	pending                            []byte
	discard                            bool
	Complete, Failed, Content, Unknown bool
}

func (s *StreamState) Feed(p []byte) {
	for len(p) > 0 {
		end := bytes.IndexByte(p, '\n')
		part := p
		if end >= 0 {
			part = p[:end]
		}
		if !s.discard {
			if len(s.pending)+len(part) <= 65536 {
				s.pending = append(s.pending, part...)
			} else {
				s.pending = nil
				s.discard = true
			}
		}
		if end < 0 {
			return
		}
		if !s.discard {
			s.line(s.pending)
		}
		s.pending = nil
		s.discard = false
		p = p[end+1:]
	}
}
func (s *StreamState) line(line []byte) {
	if !bytes.HasPrefix(line, []byte("data:")) {
		if gjson.ValidBytes(line) {
			s.control(gjson.ParseBytes(line))
		}
		return
	}
	data := bytes.TrimSpace(line[5:])
	if bytes.Equal(data, []byte("[DONE]")) {
		s.Complete = true
		return
	}
	value := gjson.ParseBytes(data)
	s.control(value)
	kind := value.Get("type").String()
	if kind == "message_stop" || kind == "response.completed" {
		s.Complete = true
	}
	if kind == "error" || kind == "response.failed" || value.Get("error").Exists() {
		s.Failed = true
	}
	if value.Get("delta.text").String() != "" || (kind == "response.output_text.delta" && value.Get("delta").String() != "") {
		s.Content = true
	}
	for _, v := range value.Get("choices").Array() {
		if v.Get("delta.content").String() != "" {
			s.Content = true
		}
	}
}
func (s *StreamState) control(value gjson.Result) {
	code := value.Get("error.code").String()
	if strings.HasSuffix(code, "_unknown") || code == "execution_outcome_unknown" {
		s.Unknown = true
	}
	if value.Get("error").Exists() {
		s.Failed = true
	}
}
func (s *StreamState) FinishJSON() {
	if len(s.pending) > 0 && !bytes.HasPrefix(s.pending, []byte("data:")) && gjson.ValidBytes(s.pending) {
		s.control(gjson.ParseBytes(s.pending))
	}
}

func Outcome(status int, stream bool, state *StreamState, cancelled, finished bool) string {
	state.FinishJSON()
	if state.Unknown {
		return "unknown"
	}
	if status >= 400 || state.Failed {
		return "failure"
	}
	if cancelled {
		return "cancelled"
	}
	if !finished || (stream && !state.Complete) {
		return "unknown"
	}
	return "success"
}
func Finish(ctx context.Context, span trace.Span, status int, stream bool, state *StreamState, cancelled, finished bool, started time.Time, fields ...any) {
	outcome := Outcome(status, stream, state, cancelled, finished)
	span.SetAttributes(attribute.String("outcome", outcome), attribute.Int("http.response.status_code", status), attribute.Bool("stream_complete", stream && state.Complete))
	if outcome == "failure" || outcome == "unknown" {
		span.SetStatus(codes.Error, outcome)
	}
	fields = append(fields, "error_code", outcomeCode(outcome))
	fields = append(fields, "outcome", outcome, "status", status, "stream_complete", stream && state.Complete, "duration_ms", float64(time.Since(started).Microseconds())/1000)
	Event(ctx, "request_completed", fields...)
	span.End()
}

func outcomeCode(outcome string) string {
	switch outcome {
	case "unknown":
		return "execution_outcome_unknown"
	case "cancelled":
		return "client_cancelled"
	case "failure":
		return "request_failed"
	}
	return ""
}

// Only explicitly configured service origins receive internal tracing credentials.
func trustedTarget(u *url.URL) bool {
	if u == nil || u.User != nil {
		return false
	}
	for _, raw := range strings.Split(os.Getenv("TRUEROUTER_TRACE_ORIGINS"), ",") {
		target, err := url.Parse(strings.TrimSpace(raw))
		if err == nil && target.User == nil && target.RawQuery == "" && target.Fragment == "" && (target.Path == "" || target.Path == "/") && target.Scheme == u.Scheme && strings.EqualFold(target.Host, u.Host) {
			return true
		}
	}
	return false
}

type transport struct{ base http.RoundTripper }

func WrapTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &transport{base}
}
func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Strip internal headers even on redirects or non-trusted targets.
	clean := req.Clone(req.Context())
	clean.Header = req.Header.Clone()
	for key := range clean.Header {
		lower := strings.ToLower(key)
		if lower == "x-observability-token" || lower == "traceparent" || lower == "tracestate" || lower == "baggage" {
			delete(clean.Header, key)
		}
	}
	req = clean
	secret := os.Getenv("TRUEROUTER_TRACE_TOKEN")
	if !trustedTarget(req.URL) || len(secret) < 32 || !trace.SpanContextFromContext(req.Context()).IsValid() {
		return t.base.RoundTrip(req)
	}
	ctx, span := tracer.Start(req.Context(), "sub2api.vm2api", trace.WithSpanKind(trace.SpanKindClient))
	request := req.Clone(ctx)
	request.Header = req.Header.Clone()
	propagation.TraceContext{}.Inject(ctx, propagation.HeaderCarrier(request.Header))
	request.Header.Set("X-Observability-Token", secret)
	started := time.Now()
	Event(ctx, "attempt_started")
	response, err := t.base.RoundTrip(request)
	if err != nil {
		Finish(ctx, span, 0, false, &StreamState{Failed: true}, ctx.Err() != nil, false, started)
		return response, err
	}
	observedUnknown := response.Header.Get("X-Observability-Outcome") == "unknown"
	response.Header.Del("X-Observability-Outcome")
	response.Body = &body{state: StreamState{Unknown: observedUnknown}, trailers: response.Trailer, ReadCloser: response.Body, ctx: ctx, span: span, status: response.StatusCode, stream: strings.Contains(response.Header.Get("Content-Type"), "text/event-stream"), started: started}
	return response, nil
}

type body struct {
	io.ReadCloser
	mu       sync.Mutex
	trailers http.Header
	ctx      context.Context
	span     trace.Span
	state    StreamState
	status   int
	stream   bool
	started  time.Time
	once     sync.Once
}

func (b *body) end(finished bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.once.Do(func() {
		if b.trailers.Get("X-Observability-Outcome") == "unknown" {
			b.state.Unknown = true
		}
		b.trailers.Del("X-Observability-Outcome")
		if b.state.Unknown {
			markUnknown(b.ctx)
		}
		Finish(b.ctx, b.span, b.status, b.stream, &b.state, b.ctx.Err() != nil, finished, b.started)
	})
}
func (b *body) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.mu.Lock()
		b.state.Feed(p[:n])
		b.mu.Unlock()
	}
	if err != nil {
		b.end(err == io.EOF)
	}
	return n, err
}
func (b *body) Close() error { err := b.ReadCloser.Close(); b.end(false); return err }
