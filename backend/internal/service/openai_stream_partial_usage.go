package service

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 客户端取消时保留已经观察到的用量；未知部分不能伪装成完整零用量。
func partialOpenAIUsage(c *gin.Context, resp *http.Response, usage *OpenAIUsage, model, upstreamModel string, started time.Time) *OpenAIForwardResult {
	if usage == nil {
		usage = &OpenAIUsage{}
	}
	return &OpenAIForwardResult{RequestID: resp.Header.Get("x-request-id"), UpstreamHeaders: resp.Header,
		Usage: *usage, Model: model, UpstreamModel: upstreamModel, Stream: true, ClientDisconnect: true,
		UpstreamResponseModel: observedUpstreamResponseModel(c), UpstreamResponseModelConflict: observedUpstreamResponseModelConflict(c), UpstreamResponseServiceTier: observedUpstreamResponseServiceTier(c),
		UsageIncomplete: boolPointer(true), Duration: time.Since(started)}
}

// 所有公开文本入口统一保留取消和用量完整性；已有解析器的终态证明优先。
func finishOpenAIStreamUsage(ctx context.Context, c *gin.Context, result *OpenAIForwardResult, err error) {
	if result == nil || !result.Stream || result.OpenAIWSMode || result.ImageEndpoint {
		return
	}
	if result.UsageIncomplete == nil {
		result.UsageIncomplete = boolPointer(err != nil || !observedUpstreamUsageComplete(c))
	}
	if ctx != nil && ctx.Err() != nil {
		result.ClientDisconnect = true
	}
}
