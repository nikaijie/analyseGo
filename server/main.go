package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"analyseGo/internal/blog"
	"analyseGo/internal/ginutil"
	"analyseGo/internal/metrics"

	"github.com/gin-gonic/gin"
)

const (
	defaultPort      = ":8099"
	defaultWindowSec = 600   // 默认10分钟
	maxWindowSec     = 86400 // 最大24小时
	sampleInterval   = time.Second
)

var (
	tracker = metrics.NewTracker()
	hub     = metrics.NewHub()
)

// handlePing 健康检查
func handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// handlePingSlow 模拟慢请求
func handlePingSlow(c *gin.Context) {
	ms := ginutil.ParseIntQuery(c, "ms", 2000)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	c.JSON(http.StatusOK, gin.H{
		"message":  "pong",
		"sleep_ms": ms,
	})
}

// handleBusy 生成大量 goroutine 模拟负载
func handleBusy(c *gin.Context) {
	n := ginutil.ParseIntQuery(c, "n", 50)
	ms := ginutil.ParseIntQuery(c, "ms", 2000)

	for i := 0; i < n; i++ {
		go func() {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}()
	}

	c.JSON(http.StatusOK, gin.H{
		"spawned":  n,
		"sleep_ms": ms,
	})
}

// handleMetrics 获取当前指标快照
func handleMetrics(c *gin.Context) {
	sample := tracker.CurrentSample()
	c.JSON(http.StatusOK, sample)
}

// handleMetricsHistory 获取历史指标数据
func handleMetricsHistory(c *gin.Context) {
	windowSec := ginutil.ParseWindowSeconds(c, maxWindowSec, defaultWindowSec)
	history := tracker.HistoryWindow(windowSec)
	c.JSON(http.StatusOK, history)
}

// handleMetricsRoutes 获取按路由统计的指标
func handleMetricsRoutes(c *gin.Context) {
	stats := tracker.RouteStats()
	c.JSON(http.StatusOK, stats)
}

// handleMetricsStream SSE 流式推送实时指标
func handleMetricsStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	// 订阅通知通道
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// 定时推送
	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			sendSample(c, tracker.CurrentSample())
		case <-ch:
			// 有新请求时立即推送
			sendSample(c, tracker.CurrentSample())
		}
	}
}

// sendSample 发送单个样本数据
func sendSample(c *gin.Context, sample metrics.Sample) {
	data, err := json.Marshal(sample)
	if err != nil {
		log.Printf("Failed to marshal sample: %v", err)
		return
	}
	fmt.Fprintf(c.Writer, "data: %s\n\n", string(data))
	c.Writer.Flush()
}

// startSampling 启动定时采样
func startSampling() {
	ticker := time.NewTicker(sampleInterval)
	go func() {
		for range ticker.C {
			tracker.PushSample()
		}
	}()
}

// setupRoutes 设置路由
func setupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		// 测试接口
		api.GET("/ping", handlePing)
		api.GET("/ping/slow", handlePingSlow)
		api.GET("/busy", handleBusy)

		// 指标接口
		api.GET("/metrics", handleMetrics)
		api.GET("/metrics/history", handleMetricsHistory)
		api.GET("/metrics/routes", handleMetricsRoutes)
		api.GET("/metrics/stream", handleMetricsStream)

		// 博客接口
		blogAPI := api.Group("/blog")
		{
			// 文章
			blogAPI.GET("/posts", blog.GetPosts)
			blogAPI.GET("/posts/:id", blog.GetPost)
			blogAPI.POST("/posts", blog.CreatePost)
			blogAPI.PUT("/posts/:id", blog.UpdatePost)
			blogAPI.DELETE("/posts/:id", blog.DeletePost)

			// 分类
			blogAPI.GET("/categories", blog.GetCategories)
			blogAPI.POST("/categories", blog.CreateCategory)

			// 标签
			blogAPI.GET("/tags", blog.GetTags)
			blogAPI.POST("/tags", blog.CreateTag)
		}
	}
}

func main() {
	// 初始化数据库
	if err := blog.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 设置 Gin 为发布模式
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(ginutil.CORSMiddleware())
	r.Use(ginutil.TrackingMiddleware(tracker, hub))

	// 设置路由
	setupRoutes(r)

	// 启动定时采样
	startSampling()

	// 启动服务器
	log.Printf("Server starting on port %s", defaultPort)
	if err := r.Run(defaultPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
