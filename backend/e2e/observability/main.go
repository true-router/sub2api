// 此夹具只启动真实入口中间件和出站传输，不连接生产数据库或账号。
package main

import (
	"context"
	logpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	tracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/observability"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--decode-otlp" || os.Args[1] == "--decode-logs") {
		raw, err := io.ReadAll(io.LimitReader(os.Stdin, 16*1024*1024))
		if err != nil {
			panic(err)
		}
		var data proto.Message = &tracepb.ExportTraceServiceRequest{}
		if os.Args[1] == "--decode-logs" {
			data = &logpb.ExportLogsServiceRequest{}
		}
		if err = proto.Unmarshal(raw, data); err != nil {
			panic(err)
		}
		out, err := protojson.Marshal(data)
		if err != nil {
			panic(err)
		}
		_, _ = os.Stdout.Write(out)
		return
	}
	shutdown := observability.Setup("e2e")
	defer shutdown()
	target, err := url.Parse(os.Getenv("E2E_TARGET"))
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(r *http.Request) { director(r); r.Header.Set("Authorization", "Bearer "+os.Getenv("E2E_API_KEY")) }
	proxy.Transport = observability.WrapTransport(http.DefaultTransport)
	proxy.FlushInterval = -1
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.RequestLogger())
	router.Any("/v1/*path", func(c *gin.Context) { proxy.ServeHTTP(c.Writer, c.Request) })
	router.GET("/health", func(c *gin.Context) { _, _ = io.WriteString(c.Writer, "ok") })
	server := &http.Server{Addr: os.Getenv("E2E_LISTEN"), Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.ListenAndServe() }()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
