// 使用真实 GatewayService 和 HTTPUpstream；账号与上游均为隔离夹具。
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/observability"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--db-check" {
		if err := checkDatabase(); err != nil {
			log.Fatal(err)
		}
		return
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "--config-check" {
		if err := json.NewEncoder(os.Stdout).Encode(cfg.Gateway); err != nil {
			log.Fatal(err)
		}
		return
	}
	cfg.Gateway.MaxLineSize = 8 * 1024 * 1024
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Security.URLAllowlist.AllowPrivateHosts = true
	upstream := repository.NewHTTPUpstream(cfg)
	gateway := service.NewGatewayService(
		nil, nil, nil, nil, nil, nil, nil, nil, cfg,
		nil, nil, nil, &service.RateLimitService{}, nil, nil,
		upstream, &service.DeferredService{}, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil)
	openaiGateway := service.NewOpenAIGatewayService(nil, nil, nil, nil, nil, nil, nil, cfg, nil, nil, nil, &service.RateLimitService{}, nil, upstream, &service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil)
	account := &service.Account{ID: 1, Name: "isolated-cancel", Platform: service.PlatformAnthropic,
		Type: service.AccountTypeAPIKey, Concurrency: 8, Status: service.StatusActive, Schedulable: true,
		Credentials: map[string]any{"api_key": os.Getenv("E2E_KEY"), "base_url": os.Getenv("E2E_TARGET")},
		Extra:       map[string]any{"anthropic_passthrough": true}}
	if os.Getenv("E2E_PROVIDER") == "openai" {
		account.Platform = service.PlatformOpenAI
		account.Extra = map[string]any{"openai_passthrough": true}
		account.Credentials["api_protocol"] = "responses"
	}
	shutdown := observability.Setup("stream-cancellation-e2e")
	defer shutdown()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), middleware.RequestLogger())
	router.GET("/health", func(c *gin.Context) { c.String(200, "ok") })
	// 失败方式：新增文本默认值误改范围外调用者的旧策略。
	// 使用正式 HTTPUpstream 检查默认旧策略与显式有界旧策略。
	router.POST("/legacy/:policy", func(c *gin.Context) {
		ctx := pkghttputil.DetachUpstreamContext(c.Request.Context())
		if c.Param("policy") == "explicit" {
			ctx = pkghttputil.DetachUpstreamContext(c.Request.Context(), time.Second)
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(400)
			return
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, os.Getenv("E2E_TARGET")+"/v1/messages", bytes.NewReader(body))
		if err != nil {
			c.Status(500)
			return
		}
		response, err := upstream.Do(request, "", 1, 8)
		if err != nil {
			c.Status(502)
			return
		}
		defer func() { _ = response.Body.Close() }()
		_, _ = io.Copy(io.Discard, response.Body)
		c.Status(204)
	})
	router.POST("/v1/*path", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(400)
			return
		}
		var fields struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if json.Unmarshal(body, &fields) != nil {
			c.Status(400)
			return
		}
		parsed := &service.ParsedRequest{Body: service.NewRequestBodyRef(body), Model: fields.Model, Stream: fields.Stream}
		var result any
		if os.Getenv("E2E_PROVIDER") == "openai" {
			switch c.Request.URL.Path {
			case "/v1/messages":
				result, err = openaiGateway.ForwardAsAnthropic(c.Request.Context(), c, account, body, "", "")
			case "/v1/chat/completions":
				result, err = openaiGateway.ForwardAsChatCompletions(c.Request.Context(), c, account, body, "", "")
			case "/v1/responses":
				result, err = openaiGateway.Forward(c.Request.Context(), c, account, body)
			default:
				c.Status(404)
				return
			}
		} else {
			switch c.Request.URL.Path {
			case "/v1/messages":
				result, err = gateway.Forward(c.Request.Context(), c, account, parsed)
			case "/v1/chat/completions":
				result, err = gateway.ForwardAsChatCompletions(c.Request.Context(), c, account, body, parsed)
			case "/v1/responses":
				result, err = gateway.ForwardAsResponses(c.Request.Context(), c, account, body, parsed)
			default:
				c.Status(404)
				return
			}
		}
		if file := os.Getenv("E2E_RECEIPT"); file != "" {
			f, e := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if e == nil {
				if err := json.NewEncoder(f).Encode(map[string]any{"path": c.Request.URL.Path, "result": result, "error": fmt.Sprint(err), "client_cancelled": c.Request.Context().Err() != nil}); err != nil {
					log.Printf("write fixture receipt: %v", err)
				}
				if err := f.Close(); err != nil {
					log.Printf("close fixture receipt: %v", err)
				}
			}
		}
		log.Printf("SYNTHETIC_FORWARD stream=%v result=%v err=%v", fields.Stream, result != nil, err)
		if err != nil && !c.Writer.Written() {
			c.JSON(502, gin.H{"error": err.Error()})
		}
	})
	log.Fatal(http.ListenAndServe(os.Getenv("E2E_LISTEN"), router))
}
