package terminalstream

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

func Serve(c *gin.Context, protocol string, expires time.Time, authorized func(context.Context) bool,
	open func(context.Context) (io.ReadCloser, int, int, int)) {
	connection, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		Subprotocols: []string{protocol},
	})
	if err != nil {
		return
	}
	defer func() { _ = connection.CloseNow() }()
	connection.SetReadLimit(32)
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	if !expires.IsZero() {
		expiry := time.AfterFunc(time.Until(expires), cancel)
		defer expiry.Stop()
	}
	if authorized != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if !authorized(ctx) {
						cancel()
						return
					}
				}
			}
		}()
	}
	credits := make(chan struct{}, 1)
	var sent, acknowledged atomic.Uint64
	go func() {
		defer cancel()
		for {
			kind, data, err := connection.Read(ctx)
			if err != nil || kind != websocket.MessageText {
				return
			}
			total, err := strconv.ParseUint(string(data), 10, 64)
			if err != nil || total < acknowledged.Load() || total > sent.Load() {
				return
			}
			acknowledged.Store(total)
			select {
			case credits <- struct{}{}:
			default:
			}
		}
	}()

	upstream, columns, rows, status := open(ctx)
	if status != http.StatusOK {
		code := websocket.StatusTryAgainLater
		if status == http.StatusNotFound {
			code = websocket.StatusPolicyViolation
		}
		_ = connection.Close(code, "terminal unavailable")
		return
	}
	defer func() { _ = upstream.Close() }()
	initial, _ := json.Marshal(gin.H{"columns": columns, "rows": rows})
	if writeTerminalFrame(ctx, connection, websocket.MessageText, initial) != nil {
		return
	}

	const maxUnacknowledged = 256 * 1024
	var buffer [16 * 1024]byte
	for {
		n, readErr := upstream.Read(buffer[:])
		if n > 0 {
			next := sent.Load() + uint64(n)
			if next-acknowledged.Load() > maxUnacknowledged {
				timer := time.NewTimer(5 * time.Second)
				for next-acknowledged.Load() > maxUnacknowledged {
					select {
					case <-credits:
					case <-timer.C:
						return
					case <-ctx.Done():
						timer.Stop()
						return
					}
				}
				timer.Stop()
			}
			sent.Store(next)
			if writeTerminalFrame(ctx, connection, websocket.MessageBinary, buffer[:n]) != nil {
				return
			}
		}
		if readErr != nil {
			_ = connection.Close(websocket.StatusNormalClosure, "")
			return
		}
	}
}

func writeTerminalFrame(ctx context.Context, connection *websocket.Conn, kind websocket.MessageType, data []byte) error {
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return connection.Write(writeCtx, kind, data)
}
