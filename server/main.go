package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Sample struct {
	Time       int64  `json:"time"`
	Goroutines int    `json:"goroutines"`
	Requests   uint64 `json:"requests"`
}

var (
	reqCounter uint64
	history    []Sample
	histMu     sync.RWMutex
)

func currentSample() Sample {
	return Sample{
		Time:       time.Now().UnixMilli(),
		Goroutines: runtime.NumGoroutine(),
		Requests:   atomic.LoadUint64(&reqCounter),
	}
}

func pushSample() {
	s := currentSample()
	histMu.Lock()
	history = append(history, s)
	if len(history) > 600 {
		history = history[len(history)-600:]
	}
	histMu.Unlock()
}

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
		atomic.AddUint64(&reqCounter, 1)
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

	api.GET("/metrics", func(c *gin.Context) {
		s := currentSample()
		c.JSON(http.StatusOK, s)
	})

	api.GET("/metrics/history", func(c *gin.Context) {
		histMu.RLock()
		h := make([]Sample, len(history))
		copy(h, history)
		histMu.RUnlock()
		c.JSON(http.StatusOK, h)
	})

	api.GET("/metrics/stream", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()

		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-c.Request.Context().Done():
				return
			case <-t.C:
				s := currentSample()
				b, _ := json.Marshal(s)
				fmt.Fprintf(c.Writer, "data: %s\n\n", string(b))
				c.Writer.Flush()
			}
		}
	})

	_ = r.Run(":8099")
}
