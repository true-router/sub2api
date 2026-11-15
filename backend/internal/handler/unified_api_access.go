package handler

import (
	"context"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// PrepareAccess reconciles an account once; a deleted default remains deleted.
func (h *APIKeyHandler) PrepareAccess(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	key, err := h.apiKeyService.EnsureUnifiedKey(c.Request.Context(), subject.UserID, true)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	state := "ready"
	var out *dto.APIKey
	if key == nil {
		state = "revoked"
	} else {
		out = dto.APIKeyFromService(key)
		if key.Status != service.StatusActive {
			state = key.Status
		}
	}
	response.Success(c, map[string]any{"state": state, "default_key": out, "billing_policy": "subscription_then_bonus_then_paid"})
}

func (h *APIKeyHandler) CreateUnifiedKey(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	executeUserIdempotentJSON(c, "user.unified_api_keys.create", struct{}{}, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		key, err := h.apiKeyService.EnsureUnifiedKey(ctx, subject.UserID, false)
		if err != nil {
			return nil, err
		}
		return dto.APIKeyFromService(key), nil
	})
}

func (h *APIKeyHandler) UnifiedModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	models, err := h.apiKeyService.UnifiedModels(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rows := make([]map[string]any, 0, len(models))
	for _, model := range models {
		offers := []map[string]any{}
		for _, route := range model.WalletRoutes {
			if h.unifiedPricing == nil {
				continue
			}
			m, timeFactor := h.unifiedPricing.PriceForUnifiedRoute(c.Request.Context(), route.Group, route.BillingModel, model.Platform)
			offers = append(offers, map[string]any{"endpoint": route.Endpoint, "rate_multiplier": route.Rate * timeFactor, "pricing": toUserPricing(m.Pricing), "time_pricing": toModelPlazaTimePricing(m.TimePricing), "long_context_basis": string(m.LongContextBasis)})
		}
		rows = append(rows, map[string]any{"id": model.ID, "platform": model.Platform, "subscription_available": model.SubscriptionAvailable, "wallet_offers": offers})
	}
	response.Success(c, rows)
}

// unifiedModels handles discovery without requiring a positive wallet balance.
func (h *GatewayHandler) unifiedModels(c *gin.Context, key *service.APIKey) bool {
	if key == nil || key.AccessMode != service.UnifiedAccessMode {
		return false
	}
	models, err := h.apiKeyService.UnifiedModels(c.Request.Context(), key.UserID)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Model discovery is unavailable")
		return true
	}
	rows := make([]map[string]any, 0, len(models))
	for _, m := range models {
		rows = append(rows, map[string]any{"id": m.ID, "object": "model", "created": 0, "owned_by": m.Platform})
	}
	c.JSON(http.StatusOK, map[string]any{"object": "list", "data": rows})
	return true
}

func ProvideAPIKeyHandler(keys *service.APIKeyService, pricing *service.ModelPlazaService) *APIKeyHandler {
	h := NewAPIKeyHandler(keys)
	h.unifiedPricing = pricing
	return h
}
