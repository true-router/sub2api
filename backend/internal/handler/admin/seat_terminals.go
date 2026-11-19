package admin

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/terminalstream"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/vmusers"
	"github.com/gin-gonic/gin"
)

func ListSeatTerminals(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	client, err := vmusers.Configured()
	if err == nil {
		seats, listErr := client.TerminalSeats(c.Request.Context(), "")
		if listErr == nil {
			response.Success(c, gin.H{"seats": seats})
			return
		}
	}
	response.Error(c, http.StatusServiceUnavailable, "席位列表暂不可用")
}

func StreamSeatTerminal(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	client, err := vmusers.Configured()
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "终端暂不可用")
		return
	}
	terminalstream.Serve(c, "sub2api-admin", time.Time{}, nil, func(ctx context.Context) (io.ReadCloser, int, int, int) {
		return client.OpenTerminal(ctx, "", c.Param("id"))
	})
}
