package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"analyseGo/internal/metrics"

	"github.com/gin-gonic/gin"
)

// Sample 与前端约定的数据结构，采用 metrics 包的类型
type Sample = metrics.Sample

var (
	tracker = metrics.NewTracker()
)

var hub = metrics.NewHub()

func addSub() chan struct{}      { return hub.Subscribe() }
func removeSub(ch chan struct{}) { hub.Unsubscribe(ch) }

func currentSample() Sample { return tracker.CurrentSample() }

func pushSample() { tracker.PushSample() }

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(func(c *gin.Context) {
		tracker.AddRequest(0)
		hub.Notify()
		c.Next()
	})

	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			pushSample()
		}
	}()

	api := r.Group("/api")
	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	api.GET("/ping/slow", func(c *gin.Context) {
		msStr := c.Query("ms")
		ms, _ := strconv.Atoi(msStr)
		if ms <= 0 {
			ms = 2000
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "pong", "sleep_ms": ms})
	})

	api.GET("/busy", func(c *gin.Context) {
		nStr := c.Query("n")
		msStr := c.Query("ms")
		n, _ := strconv.Atoi(nStr)
		if n <= 0 {
			n = 50
		}
		ms, _ := strconv.Atoi(msStr)
		if ms <= 0 {
			ms = 2000
		}
		for i := 0; i < n; i++ {
			go func() { time.Sleep(time.Duration(ms) * time.Millisecond) }()
		}
		c.JSON(http.StatusOK, gin.H{"spawned": n, "sleep_ms": ms})
	})

	api.GET("/metrics", func(c *gin.Context) { c.JSON(http.StatusOK, currentSample()) })

	api.GET("/metrics/history", func(c *gin.Context) { c.JSON(http.StatusOK, tracker.History()) })

	api.GET("/metrics/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		ch := addSub()
		defer removeSub(ch)

		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-t.C:
				b, _ := json.Marshal(currentSample())
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
				c.Writer.Flush()
			case <-ch:
				b, _ := json.Marshal(currentSample())
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
				c.Writer.Flush()
			}
		}
	})

	_ = r.Run(":8099")
}
