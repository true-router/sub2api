package rollout

import (
	"bufio"
	"crypto/subtle"
	"github.com/gin-gonic/gin"
	"net"
	"os"
	"strings"
	"sync"
)

// Controller keeps old requests alive until the publisher verifies complete drain.
type Controller struct {
	mu       sync.Mutex
	draining bool
	active   int
	token    string
	instance string
	pending  func() uint64
}

func FromEnvironment(pending ...func() uint64) gin.HandlerFunc {
	file := os.Getenv("TRUEROUTER_ROLLOUT_TOKEN_FILE")
	if file == "" {
		return func(c *gin.Context) { c.Next() }
	}
	stat, err := os.Lstat(file)
	if err != nil || !stat.Mode().IsRegular() || stat.Mode().Perm()&0077 != 0 {
		panic("private rollout token file required")
	}
	raw, err := os.ReadFile(file)
	token := strings.TrimSpace(string(raw))
	instance := os.Getenv("TRUEROUTER_INSTANCE_ID")
	if err != nil || len(token) < 32 || instance == "" {
		panic("rollout token and instance required")
	}
	controller := &Controller{token: token, instance: instance}
	if len(pending) > 0 {
		controller.pending = pending[0]
	}
	return controller.middleware
}

func (d *Controller) snapshot() gin.H {
	d.mu.Lock()
	defer d.mu.Unlock()
	pending := uint64(0)
	if d.pending != nil {
		pending = d.pending()
	}
	return gin.H{"schema": 1, "instance": d.instance, "accepting": !d.draining, "active_requests": d.active, "pending_usage_records": pending, "drained": d.draining && d.active == 0 && pending == 0}
}

func (d *Controller) middleware(c *gin.Context) {
	p := c.Request.URL.Path
	if strings.HasPrefix(p, "/_rollout/") {
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte("Bearer "+d.token)) != 1 {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		switch {
		case p == "/_rollout/status" && c.Request.Method == "GET":
		case p == "/_rollout/drain" && c.Request.Method == "POST":
			d.mu.Lock()
			d.draining = true
			d.mu.Unlock()
		case p == "/_rollout/resume" && c.Request.Method == "POST":
			d.mu.Lock()
			d.draining = false
			d.mu.Unlock()
		default:
			c.AbortWithStatus(404)
			return
		}
		c.AbortWithStatusJSON(200, d.snapshot())
		return
	}
	d.mu.Lock()
	if d.draining {
		d.mu.Unlock()
		c.Header("Connection", "close")
		c.AbortWithStatusJSON(503, gin.H{"error": "instance_draining"})
		return
	}
	d.active++
	d.mu.Unlock()
	var once sync.Once
	release := func() { once.Do(func() { d.mu.Lock(); d.active--; d.mu.Unlock() }) }
	writer := &trackedWriter{ResponseWriter: c.Writer, release: release}
	c.Writer = writer
	defer func() {
		if !writer.hijacked {
			release()
		}
	}()
	c.Next()
}

type trackedWriter struct {
	gin.ResponseWriter
	release  func()
	hijacked bool
}

func (w *trackedWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	conn, rw, err := w.ResponseWriter.Hijack()
	if err != nil {
		return conn, rw, err
	}
	w.hijacked = true
	return &trackedConn{Conn: conn, release: w.release}, rw, nil
}

type trackedConn struct {
	net.Conn
	release func()
}

func (c *trackedConn) Close() error { defer c.release(); return c.Conn.Close() }
