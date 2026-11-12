package middleware

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const requestIDHeader = "X-Request-ID"

// RequestLogger 在请求入口注入 request-scoped logger。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil {
			c.Next()
			return
		}

		requestID, validRequestID := normalizeCorrelationID(c.GetHeader(requestIDHeader))
		if !validRequestID {
			requestID = uuid.NewString()
		}
		c.Header(requestIDHeader, requestID)

		ctx, span := observability.StartRequest(c.Request.Context())
		span.SetAttributes(attribute.String("request_id", requestID))
		started := time.Now()
		state := &observability.StreamState{}
		c.Writer = &traceResponseWriter{ResponseWriter: c.Writer, state: state, ctx: ctx}
		defer func() {
			state.Unknown = state.Unknown || observability.RequestUnknown(ctx)
			observability.Finish(ctx, span, c.Writer.Status(), strings.Contains(c.Writer.Header().Get("Content-Type"), "text/event-stream"), state, ctx.Err() != nil, true, started, "request_id", requestID, "response_started", c.Writer.Size() > 0)
		}()
		ctx = context.WithValue(ctx, ctxkey.RequestID, requestID)
		clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string)
		clientRequestID, _ = normalizeCorrelationID(clientRequestID)

		requestLogger := logger.With(
			zap.String("component", "http"),
			zap.String("trace_id", trace.SpanContextFromContext(ctx).TraceID().String()),
			zap.String("span_id", trace.SpanContextFromContext(ctx).SpanID().String()),
			zap.String("request_id", requestID),
			zap.String("client_request_id", strings.TrimSpace(clientRequestID)),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)

		ctx = logger.IntoContext(ctx, requestLogger)
		c.Request = c.Request.WithContext(ctx)
		observability.Event(ctx, "request_started", "request_id", requestID)
		c.Next()
	}
}

// Keep Gin interfaces, SSE flushing and the original response bytes intact.
type traceResponseWriter struct {
	gin.ResponseWriter
	state *observability.StreamState
	ctx   context.Context
}

func (w *traceResponseWriter) Write(p []byte) (int, error) {
	n, e := w.ResponseWriter.Write(p)
	was := w.state.Content
	w.state.Feed(p[:n])
	if !was && w.state.Content {
		observability.Event(w.ctx, "first_content")
	}
	return n, e
}
func (w *traceResponseWriter) WriteString(p string) (int, error) { return w.Write([]byte(p)) }
