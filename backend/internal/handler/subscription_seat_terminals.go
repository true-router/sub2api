package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/terminalstream"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/vmusers"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func (h *SubscriptionHandler) ListSeatTerminals(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "需要登录")
		return
	}
	client, err := vmusers.Configured()
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "席位列表暂不可用")
		return
	}
	userID, _, err := h.subscriptionService.TerminalSubscription(c.Request.Context(), subject.UserID, client.Origin)
	if err == nil && userID == "" {
		response.Success(c, gin.H{"seats": []vmusers.Seat{}})
		return
	}
	if err == nil {
		if seats, listErr := client.TerminalSeats(c.Request.Context(), userID); listErr == nil {
			response.Success(c, gin.H{"seats": seats})
			return
		}
	}
	response.Error(c, http.StatusServiceUnavailable, "席位列表暂不可用")
}

func (h *SubscriptionHandler) StreamSeatTerminal(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "需要登录")
		return
	}
	client, err := vmusers.Configured()
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "终端暂不可用")
		return
	}
	userID, expires, err := h.subscriptionService.TerminalSubscription(c.Request.Context(), subject.UserID, client.Origin)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "终端暂不可用")
		return
	}
	if userID == "" {
		response.Error(c, http.StatusNotFound, "席位不可用")
		return
	}
	id := c.Param("id")
	terminalstream.Serve(c, "sub2api-terminal", expires, func(ctx context.Context) bool {
		current, _, err := h.subscriptionService.TerminalSubscription(ctx, subject.UserID, client.Origin)
		return err == nil && current == userID
	}, func(ctx context.Context) (io.ReadCloser, int, int, int) {
		return client.OpenTerminal(ctx, userID, id)
	})
}
