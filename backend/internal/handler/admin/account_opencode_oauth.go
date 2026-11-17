package admin

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strings"
)

func (h *AccountHandler) SetOpenCodeOAuthService(s *service.OpenCodeOAuthService) {
	h.openCodeOAuth = s
}

type openCodeOAuthRequest struct {
	ProxyID    *int64          `json:"proxy_id"`
	DeviceCode string          `json:"device_code"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AccountHandler) OpenCodeAuthorize(c *gin.Context) {
	var req openCodeOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if h.openCodeOAuth == nil {
		response.Error(c, 503, "OpenCode OAuth is unavailable")
		return
	}
	device, err := h.openCodeOAuth.Authorize(c.Request.Context(), req.ProxyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, device)
}

func (h *AccountHandler) OpenCodePoll(c *gin.Context) {
	var req openCodeOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.DeviceCode) == "" {
		response.BadRequest(c, "device_code is required")
		return
	}
	if h.openCodeOAuth == nil {
		response.Error(c, 503, "OpenCode OAuth is unavailable")
		return
	}
	credentials, status, err := h.openCodeOAuth.Poll(c.Request.Context(), req.ProxyID, req.DeviceCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": status, "credentials": credentials})
}

func (h *AccountHandler) OpenCodeImport(c *gin.Context) {
	var req openCodeOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if h.openCodeOAuth == nil {
		response.Error(c, 503, "OpenCode OAuth is unavailable")
		return
	}
	credentials, err := h.openCodeOAuth.Import(req.Credential)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"status": "authorized", "credentials": credentials})
}
