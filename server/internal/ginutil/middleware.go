package ginutil

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"analyseGo/internal/metrics"
	pprof "runtime/pprof"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 处理跨域请求的中间件
func CORSMiddleware() gin.HandlerFunc {
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

// TrackingMiddleware 追踪请求并添加 pprof 标签的中间件
// 用于记录请求路由和添加性能分析标签
func TrackingMiddleware(tracker *metrics.Tracker, hub *metrics.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		route := GetRoutePath(c)

		// 记录请求开始时间（用于计算CPU时间）
		startTime := time.Now()

		// 记录请求前的内存
		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// 记录请求
		tracker.AddRequest(0)
		tracker.AddRequestRoute(route, 0)
		hub.Notify()

		// 添加 pprof 标签用于性能分析
		labels := pprof.Labels("route", route)
		pprof.Do(c.Request.Context(), labels, func(ctx context.Context) {
			c.Request = c.Request.WithContext(ctx)
			c.Next()

			// 请求完成后记录内存增量和CPU时间
			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			// 计算内存增量（使用 HeapAlloc 的变化）
			if memAfter.HeapAlloc > memBefore.HeapAlloc {
				memDelta := memAfter.HeapAlloc - memBefore.HeapAlloc
				tracker.AddRouteMemory(route, memDelta)
			}

			// 计算CPU时间（纳秒）
			cpuTimeNs := time.Since(startTime).Nanoseconds()
			tracker.AddRouteCPUTime(route, cpuTimeNs)
		})
	}
}
