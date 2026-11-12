package main

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/rollout"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"time"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	pool := service.NewUsageRecordWorkerPoolWithOptions(service.UsageRecordWorkerPoolOptions{WorkerCount: 1, QueueSize: 4, TaskTimeout: 5 * time.Second})
	defer pool.Stop()
	pending := func() uint64 {
		s := pool.Stats()
		if s.CompletedTasks > s.SubmittedTasks {
			return 1
		}
		return s.SubmittedTasks - s.CompletedTasks
	}
	r.Use(rollout.FromEnvironment(pending))
	r.GET("/billing", func(c *gin.Context) {
		pool.Submit(func(ctx context.Context) { time.Sleep(1700 * time.Millisecond) })
		c.JSON(200, gin.H{"queued": true})
	})
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/normal", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.GET("/slow", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.String(200, "data: started\n\n")
		c.Writer.Flush()
		time.Sleep(time.Second)
		c.String(200, "data: completed\n\n")
		c.Writer.Flush()
	})
	if err := http.ListenAndServe(os.Getenv("E2E_LISTEN"), r); err != nil {
		panic(err)
	}
}
