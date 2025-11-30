package ginutil

import "github.com/gin-gonic/gin"

// GetRoutePath 提取路由路径
// 优先使用 FullPath()，如果为空则使用 Request.URL.Path
func GetRoutePath(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}
	return route
}
