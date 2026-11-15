package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

func (h *AnnouncementHandler) ListNotifications(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	items, err := h.notifications.List(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}
func (h *AnnouncementHandler) ReadNotification(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid notification ID")
		return
	}
	if err = h.notifications.MarkRead(c.Request.Context(), subject.UserID, c.Param("type"), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "ok"})
}

func ProvideAnnouncementHandler(announcements *service.AnnouncementService, notifications *service.NotificationService) *AnnouncementHandler {
	h := NewAnnouncementHandler(announcements)
	h.notifications = notifications
	return h
}
