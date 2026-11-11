package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// OPCPay only accepts an action ID and SKU. Identity and amounts are server-owned.
func (h *PaymentHandler) OPCPayInfo(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	result, err := h.paymentService.OPCPay().Info(c.Request.Context(), subject.UserID)
	if err != nil {
		response.Error(c, 502, err.Error())
		return
	}
	response.Success(c, result)
}
func (h *PaymentHandler) OPCPayStart(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	var body struct {
		Kind   string `json:"kind"`
		Action string `json:"action_id"`
		SKU    string `json:"sku_id"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		response.BadRequest(c, "请求参数无效")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.BadRequest(c, "请求参数无效")
		return
	}
	c.Header("Cache-Control", "private, no-store")
	result, err := h.paymentService.OPCPay().Start(c.Request.Context(), subject.UserID, body.Kind, body.Action, body.SKU)
	if err != nil {
		response.Error(c, 409, err.Error())
		return
	}
	response.Success(c, result)
}
func (h *PaymentHandler) OPCPayResume(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "private, no-store")
	result, err := h.paymentService.OPCPay().Resume(c.Request.Context(), subject.UserID, c.Param("business_id"))
	if err != nil {
		response.Error(c, 409, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *PaymentHandler) OPCPayClosePurchase(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	if err := h.paymentService.OPCPay().ClosePurchase(c.Request.Context(), subject.UserID, c.Param("business_id")); err != nil {
		response.Error(c, 409, err.Error())
		return
	}
	response.Success(c, gin.H{"closed": true})
}
