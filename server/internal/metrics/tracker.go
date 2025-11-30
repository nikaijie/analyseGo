package metrics

import (
	"bytes"
	"regexp"
	"runtime"
	pprof "runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxHistory 最多保留的历史样本数量（86400秒 = 24小时，每秒一个样本）
	maxHistory = 86400
	// requestWindowDuration 统计请求数的时间窗口
	requestWindowDuration = 10 * time.Second
)

// Sample 指标样本数据结构
type Sample struct {
	Time        int64  `json:"time"`        // 时间戳（毫秒）
	Goroutines  int    `json:"goroutines"`  // Goroutine 数量
	Requests    int    `json:"requests"`    // 最近10秒的请求数
	HeapAlloc   uint64 `json:"heapAlloc"`   // 堆内存已分配（字节）
	HeapInuse   uint64 `json:"heapInuse"`   // 堆内存使用中（字节）
	HeapSys     uint64 `json:"heapSys"`     // 堆内存系统占用（字节）
	HeapObjects uint64 `json:"heapObjects"` // 堆对象数量
	NumGC       uint32 `json:"numGC"`       // GC次数（累计）
	GCIncrement uint32 `json:"gcIncrement"` // 本次采样期间的GC增量
	BlockLock   int    `json:"blockLock"`   // 锁阻塞的 goroutine 数量
	BlockIO     int    `json:"blockIO"`     // IO 阻塞的 goroutine 数量
	BlockPerm   int    `json:"blockPerm"`   // 持续≥10秒的阻塞 goroutine 数量
}

// Tracker 负责指标采样和请求统计
type Tracker struct {
	mu sync.RWMutex
	// reqTimes 记录每次请求到达的时间戳（毫秒）
	reqTimes []int64
	// reqByRoute 按路由记录最近请求时间戳（毫秒）
	reqByRoute map[string][]int64

	histMu            sync.RWMutex
	history           []Sample          // 历史样本数据
	lastNumGC         uint32            // 上一次采样的GC次数（用于计算增量）
	routeMemory       map[string]uint64 // 按路由记录的内存使用（字节）
	routeRequestCount map[string]uint64 // 按路由记录的总请求数（用于计算平均内存）
	routeCPUTime      map[string]int64  // 按路由记录的CPU时间（纳秒）
	lastMemStats      runtime.MemStats  // 上一次的内存统计
}

// NewTracker 创建新的追踪器实例
func NewTracker() *Tracker {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return &Tracker{
		reqByRoute:        make(map[string][]int64),
		routeMemory:       make(map[string]uint64),
		routeRequestCount: make(map[string]uint64),
		routeCPUTime:      make(map[string]int64),
		lastMemStats:      ms,
	}
}

// AddRequest 记录一次请求到达
func (t *Tracker) AddRequest(ts int64) {
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	t.mu.Lock()
	t.reqTimes = append(t.reqTimes, ts)
	t.mu.Unlock()
}

// AddRequestRoute 记录某路由一次请求到达
func (t *Tracker) AddRequestRoute(route string, ts int64) {
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	t.mu.Lock()
	if t.reqByRoute == nil {
		t.reqByRoute = make(map[string][]int64)
	}
	t.reqByRoute[route] = append(t.reqByRoute[route], ts)
	t.mu.Unlock()
}

// AddRouteMemory 记录某路由的内存使用增量
func (t *Tracker) AddRouteMemory(route string, memDelta uint64) {
	t.mu.Lock()
	if t.routeMemory == nil {
		t.routeMemory = make(map[string]uint64)
	}
	if t.routeRequestCount == nil {
		t.routeRequestCount = make(map[string]uint64)
	}
	t.routeMemory[route] += memDelta
	t.routeRequestCount[route]++
	t.mu.Unlock()
}

// AddRouteCPUTime 记录某路由的CPU时间（纳秒）
func (t *Tracker) AddRouteCPUTime(route string, cpuTimeNs int64) {
	t.mu.Lock()
	if t.routeCPUTime == nil {
		t.routeCPUTime = make(map[string]int64)
	}
	t.routeCPUTime[route] += cpuTimeNs
	t.mu.Unlock()
}

// requestsInWindow 统计最近 duration 内的请求数，并清理过期数据
func (t *Tracker) requestsInWindow(duration time.Duration) int {
	cutoff := time.Now().Add(-duration).UnixMilli()
	t.mu.Lock()
	defer t.mu.Unlock()

	// 找到第一个未过期的时间戳
	i := 0
	for ; i < len(t.reqTimes); i++ {
		if t.reqTimes[i] >= cutoff {
			break
		}
	}

	// 清理过期数据
	if i > 0 {
		t.reqTimes = t.reqTimes[i:]
	}

	return len(t.reqTimes)
}

// requestsInWindowByRoute 统计最近 duration 内每个路由的请求数，并清理过期数据
func (t *Tracker) requestsInWindowByRoute(duration time.Duration) map[string]int {
	cutoff := time.Now().Add(-duration).UnixMilli()
	t.mu.Lock()
	defer t.mu.Unlock()

	res := make(map[string]int, len(t.reqByRoute))
	for route, times := range t.reqByRoute {
		// 找到第一个未过期的时间戳
		i := 0
		for ; i < len(times); i++ {
			if times[i] >= cutoff {
				break
			}
		}

		// 清理过期数据
		if i > 0 {
			times = times[i:]
			t.reqByRoute[route] = times
		}

		res[route] = len(times)
	}

	return res
}

// CurrentSample 返回当前时刻的指标采样
func (t *Tracker) CurrentSample() Sample {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	blockLock, blockIO, blockPerm := classifyBlocks()

	// 计算GC增量
	currentNumGC := ms.NumGC
	var gcIncrement uint32
	if currentNumGC >= t.lastNumGC {
		gcIncrement = currentNumGC - t.lastNumGC
	} else {
		// GC次数重置（理论上不应该发生，但处理溢出情况）
		gcIncrement = currentNumGC
	}
	t.lastNumGC = currentNumGC

	return Sample{
		Time:        time.Now().UnixMilli(),
		Goroutines:  runtime.NumGoroutine(),
		Requests:    t.requestsInWindow(requestWindowDuration),
		HeapAlloc:   ms.HeapAlloc,
		HeapInuse:   ms.HeapInuse,
		HeapSys:     ms.HeapSys,
		HeapObjects: ms.HeapObjects,
		NumGC:       currentNumGC,
		GCIncrement: gcIncrement,
		BlockLock:   blockLock,
		BlockIO:     blockIO,
		BlockPerm:   blockPerm,
	}
}

// classifyBlocks 分析并分类当前 goroutine 的阻塞状态
// 返回: (锁阻塞数, IO阻塞数, 持续≥10秒阻塞数)
func classifyBlocks() (lock int, io int, perm int) {
	var buf bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&buf, 1)
	lines := strings.Split(buf.String(), "\n")

	// 匹配 goroutine 状态行: "goroutine 123 [state]:"
	re := regexp.MustCompile(`goroutine\s+\d+\s+\[(.*?)\]:`)
	// 匹配持续时间: "123 minute(s)" 或 "123 second(s)"
	durRe := regexp.MustCompile(`(\d+)\s*(minute|minutes|second|seconds)`)

	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}

		state := strings.ToLower(m[1])
		var durSec int

		// 解析持续时间
		if dm := durRe.FindStringSubmatch(state); len(dm) >= 3 {
			n := dm[1]
			unit := dm[2]
			if v, err := strconv.Atoi(n); err == nil {
				if strings.HasPrefix(unit, "minute") {
					durSec = v * 60
				} else {
					durSec = v
				}
			}
		}

		// 分类阻塞类型
		if strings.Contains(state, "semacquire") {
			lock++
		}
		if strings.Contains(state, "io wait") || strings.Contains(state, "syscall") {
			io++
		}
		// 持续≥10秒的阻塞
		if durSec >= 10 {
			if strings.Contains(state, "semacquire") ||
				strings.Contains(state, "chan receive") ||
				strings.Contains(state, "chan send") ||
				strings.Contains(state, "select") ||
				strings.Contains(state, "io wait") ||
				strings.Contains(state, "syscall") {
				perm++
			}
		}
	}

	return
}

// classifyBlocksByRoute 按路由标签分类统计阻塞状态
// 返回: map[route][锁阻塞数, IO阻塞数, 持续≥10秒阻塞数]
func classifyBlocksByRoute() map[string][3]int {
	var buf bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&buf, 2) // 使用深度2以包含标签信息
	lines := strings.Split(buf.String(), "\n")

	reHeader := regexp.MustCompile(`goroutine\s+\d+\s+\[(.*?)\]:`)
	reLabels := regexp.MustCompile(`labels:\s+(.*)`)
	reRouteKV := regexp.MustCompile(`route=([^,\s]+)`)
	reDur := regexp.MustCompile(`(\d+)\s*(minute|minutes|second|seconds)`)

	stats := make(map[string][3]int)
	i := 0

	for i < len(lines) {
		m := reHeader.FindStringSubmatch(lines[i])
		if len(m) < 2 {
			i++
			continue
		}

		state := strings.ToLower(m[1])
		var durSec int

		// 解析持续时间
		if dm := reDur.FindStringSubmatch(state); len(dm) >= 3 {
			n := dm[1]
			unit := dm[2]
			if v, err := strconv.Atoi(n); err == nil {
				if strings.HasPrefix(unit, "minute") {
					durSec = v * 60
				} else {
					durSec = v
				}
			}
		}

		// 查找路由标签
		route := ""
		j := i + 1
		for j < len(lines) && !reHeader.MatchString(lines[j]) {
			if lm := reLabels.FindStringSubmatch(lines[j]); len(lm) >= 2 {
				if rm := reRouteKV.FindStringSubmatch(lm[1]); len(rm) >= 2 {
					route = rm[1]
					break // 找到路由后立即退出
				}
			}
			j++
		}

		// 如果没有找到路由标签，跳过这个goroutine（不统计到unknown）
		if route == "" {
			i = j
			continue
		}

		// 统计阻塞类型
		var a, b, c int
		if strings.Contains(state, "semacquire") {
			a = 1
		}
		if strings.Contains(state, "io wait") || strings.Contains(state, "syscall") {
			b = 1
		}
		if durSec >= 10 {
			if strings.Contains(state, "semacquire") ||
				strings.Contains(state, "chan receive") ||
				strings.Contains(state, "chan send") ||
				strings.Contains(state, "select") ||
				strings.Contains(state, "io wait") ||
				strings.Contains(state, "syscall") {
				c = 1
			}
		}

		cur := stats[route]
		stats[route] = [3]int{cur[0] + a, cur[1] + b, cur[2] + c}
		i = j
	}

	return stats
}

// RouteStat 路由统计信息
type RouteStat struct {
	Route       string  `json:"route"`       // 路由路径
	Requests    int     `json:"requests"`    // 最近10秒的请求数
	MemoryUsage float64 `json:"memoryUsage"` // 内存消耗（MB），请求数 × 平均每个请求的内存
	CPUUsage    float64 `json:"cpuUsage"`    // CPU消耗（ms），当前窗口内的CPU时间
	BlockLock   int     `json:"blockLock"`   // 锁阻塞数
	BlockIO     int     `json:"blockIO"`     // IO 阻塞数
	BlockPerm   int     `json:"blockPerm"`   // 持续≥10秒阻塞数
}

// RouteStats 获取按路由统计的指标
func (t *Tracker) RouteStats() []RouteStat {
	reqs := t.requestsInWindowByRoute(requestWindowDuration)
	blocks := classifyBlocksByRoute()

	t.mu.RLock()
	routeMem := make(map[string]uint64)
	routeReqCount := make(map[string]uint64)
	routeCPU := make(map[string]int64)
	for r, m := range t.routeMemory {
		routeMem[r] = m
	}
	for r, c := range t.routeRequestCount {
		routeReqCount[r] = c
	}
	for r, cpu := range t.routeCPUTime {
		routeCPU[r] = cpu
	}
	t.mu.RUnlock()

	// 收集所有路由（排除空路由和unknown）
	routes := make(map[string]struct{})
	for r := range reqs {
		if r != "" && r != "(unknown)" {
			routes[r] = struct{}{}
		}
	}
	for r := range blocks {
		if r != "" && r != "(unknown)" {
			routes[r] = struct{}{}
		}
	}

	// 构建结果
	out := make([]RouteStat, 0, len(routes))
	for r := range routes {
		b := blocks[r]
		requestCount := reqs[r]

		// 计算内存消耗
		// 方法：总内存 / 总请求数 = 平均每个请求的内存
		// 当前窗口内的内存消耗 = 平均内存 × 当前窗口请求数
		var memoryUsage float64
		totalMem := routeMem[r]
		totalReqCount := routeReqCount[r]
		if requestCount > 0 && totalMem > 0 && totalReqCount > 0 {
			// 计算平均每个请求的内存（MB）
			avgMemPerRequest := float64(totalMem) / 1024 / 1024 / float64(totalReqCount)
			// 当前窗口内的内存消耗 = 平均内存 × 当前窗口请求数
			memoryUsage = avgMemPerRequest * float64(requestCount)
		}

		// 计算CPU消耗（毫秒）
		// 方法：总CPU时间 / 总请求数 = 平均每个请求的CPU时间
		// 当前窗口内的CPU消耗 = 平均CPU时间 × 当前窗口请求数
		var cpuUsage float64
		totalCPUTime := routeCPU[r]
		if requestCount > 0 && totalCPUTime > 0 && totalReqCount > 0 {
			// 计算平均每个请求的CPU时间（毫秒）
			avgCPUTimePerRequest := float64(totalCPUTime) / 1e6 / float64(totalReqCount) // 纳秒转毫秒
			// 当前窗口内的CPU消耗 = 平均CPU时间 × 当前窗口请求数
			cpuUsage = avgCPUTimePerRequest * float64(requestCount)
		}

		out = append(out, RouteStat{
			Route:       r,
			Requests:    requestCount,
			MemoryUsage: memoryUsage,
			CPUUsage:    cpuUsage,
			BlockLock:   b[0],
			BlockIO:     b[1],
			BlockPerm:   b[2],
		})
	}

	return out
}

// PushSample 将当前样本推入历史，最多保留 maxHistory 个点
func (t *Tracker) PushSample() {
	s := t.CurrentSample()
	t.histMu.Lock()
	t.history = append(t.history, s)
	if len(t.history) > maxHistory {
		t.history = t.history[len(t.history)-maxHistory:]
	}
	t.histMu.Unlock()
}

// History 返回完整历史数据的拷贝
func (t *Tracker) History() []Sample {
	t.histMu.RLock()
	defer t.histMu.RUnlock()

	h := make([]Sample, len(t.history))
	copy(h, t.history)
	return h
}

// HistoryWindow 返回指定时间窗口内的历史数据（秒）
// 如果 seconds <= 0，默认返回最近10分钟的数据
func (t *Tracker) HistoryWindow(seconds int) []Sample {
	if seconds <= 0 {
		seconds = 600 // 默认10分钟
	}

	cutoff := time.Now().Add(-time.Duration(seconds) * time.Second).UnixMilli()

	t.histMu.RLock()
	defer t.histMu.RUnlock()

	h := t.history
	if len(h) == 0 {
		return []Sample{}
	}

	// 从后往前查找第一个未过期的时间戳
	idx := 0
	for i := len(h) - 1; i >= 0; i-- {
		if h[i].Time < cutoff {
			idx = i + 1
			break
		}
	}

	// 返回窗口内的数据
	out := make([]Sample, len(h)-idx)
	copy(out, h[idx:])
	return out
}
