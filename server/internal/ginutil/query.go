package ginutil

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParseIntQuery 解析整数查询参数
// 如果参数不存在、解析失败或值小于等于0，则返回默认值
func ParseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valStr := c.Query(key)
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil || val <= 0 {
		return defaultValue
	}
	return val
}

// ParseWindowSeconds 解析时间窗口参数（支持 window/minutes/hours）
// 参数优先级：window > minutes > hours
// 如果所有参数都无效，返回 defaultWindowSec
// 如果解析的值超过 maxWindowSec，则限制为 maxWindowSec
func ParseWindowSeconds(c *gin.Context, maxWindowSec, defaultWindowSec int) int {
	// 优先使用 window 参数（秒）
	if wStr := c.Query("window"); wStr != "" {
		if sec, err := strconv.Atoi(wStr); err == nil && sec > 0 {
			if sec > maxWindowSec {
				sec = maxWindowSec
			}
			return sec
		}
	}

	// 尝试 minutes 参数
	if mStr := c.Query("minutes"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil && m > 0 {
			sec := m * 60
			if sec > maxWindowSec {
				sec = maxWindowSec
			}
			return sec
		}
	}

	// 尝试 hours 参数
	if hStr := c.Query("hours"); hStr != "" {
		if h, err := strconv.Atoi(hStr); err == nil && h > 0 {
			sec := h * 3600
			if sec > maxWindowSec {
				sec = maxWindowSec
			}
			return sec
		}
	}

	return defaultWindowSec
}
