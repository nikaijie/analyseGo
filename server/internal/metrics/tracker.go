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

type Sample struct {
	Time        int64  `json:"time"`
	Goroutines  int    `json:"goroutines"`
	Requests    int    `json:"requests"`
	HeapAlloc   uint64 `json:"heapAlloc"`
	HeapInuse   uint64 `json:"heapInuse"`
	HeapSys     uint64 `json:"heapSys"`
	HeapObjects uint64 `json:"heapObjects"`
	BlockLock   int    `json:"blockLock"`
	BlockIO     int    `json:"blockIO"`
	BlockPerm   int    `json:"blockPerm"`
}

// Tracker 负责采样与“最近10秒请求数”的统计
type Tracker struct {
	mu sync.RWMutex
	// 记录每次请求到达的时间戳（毫秒）
	reqTimes []int64

	// 按路由记录最近请求时间戳（毫秒）
	reqByRoute map[string][]int64

	histMu  sync.RWMutex
	history []Sample
}

func NewTracker() *Tracker {
	return &Tracker{reqByRoute: make(map[string][]int64)}
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
	t.reqByRoute[route] = append(t.reqByRoute[route], ts)
	t.mu.Unlock()
}

// requestsInWindow 统计最近 duration 内的请求数，并顺便清理过期数据
func (t *Tracker) requestsInWindow(duration time.Duration) int {
	cutoff := time.Now().Add(-duration).UnixMilli()
	t.mu.Lock()
	// 移除早于 cutoff 的时间戳
	i := 0
	for ; i < len(t.reqTimes); i++ {
		if t.reqTimes[i] >= cutoff {
			break
		}
	}
	if i > 0 {
		t.reqTimes = t.reqTimes[i:]
	}
	count := len(t.reqTimes)
	t.mu.Unlock()
	return count
}

// requestsInWindowByRoute 统计最近 duration 每个路由的请求数，并清理过期数据
func (t *Tracker) requestsInWindowByRoute(duration time.Duration) map[string]int {
	cutoff := time.Now().Add(-duration).UnixMilli()
	t.mu.Lock()
	res := make(map[string]int, len(t.reqByRoute))
	for r, times := range t.reqByRoute {
		i := 0
		for ; i < len(times); i++ {
			if times[i] >= cutoff {
				break
			}
		}
		if i > 0 {
			times = times[i:]
		}
		t.reqByRoute[r] = times
		res[r] = len(times)
	}
	t.mu.Unlock()
	return res
}

// CurrentSample 返回当前采样
func (t *Tracker) CurrentSample() Sample {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	bl, bi, bp := classifyBlocks()
	return Sample{
		Time:        time.Now().UnixMilli(),
		Goroutines:  runtime.NumGoroutine(),
		Requests:    t.requestsInWindow(10 * time.Second),
		HeapAlloc:   ms.HeapAlloc,
		HeapInuse:   ms.HeapInuse,
		HeapSys:     ms.HeapSys,
		HeapObjects: ms.HeapObjects,
		BlockLock:   bl,
		BlockIO:     bi,
		BlockPerm:   bp,
	}
}

func classifyBlocks() (lock int, io int, perm int) {
	var buf bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&buf, 1)
	lines := strings.Split(buf.String(), "\n")
	re := regexp.MustCompile(`goroutine\s+\d+\s+\[(.*?)\]:`)
	durRe := regexp.MustCompile(`(\d+)\s*(minute|minutes|second|seconds)`)
	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) < 2 {
			continue
		}
		state := strings.ToLower(m[1])
		var durSec int
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
		if strings.Contains(state, "semacquire") {
			lock++
		}
		if strings.Contains(state, "io wait") || strings.Contains(state, "syscall") {
			io++
		}
		if durSec >= 10 {
			if strings.Contains(state, "semacquire") || strings.Contains(state, "chan receive") || strings.Contains(state, "chan send") || strings.Contains(state, "select") || strings.Contains(state, "io wait") || strings.Contains(state, "syscall") {
				perm++
			}
		}
	}
	return
}

// classifyBlocksByRoute 返回按 route 标签统计的阻塞分类
func classifyBlocksByRoute() map[string][3]int {
	var buf bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&buf, 2)
	lines := strings.Split(buf.String(), "\n")
	reHeader := regexp.MustCompile(`goroutine\s+\d+\s+\[(.*?)\]:`)
	reLabels := regexp.MustCompile(`labels:\s+(.*)`)
	reRouteKV := regexp.MustCompile(`route=([^,\s]+)`) // route 值以空格或逗号分隔
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
		route := "(unknown)"
		j := i + 1
		for j < len(lines) && !reHeader.MatchString(lines[j]) {
			if lm := reLabels.FindStringSubmatch(lines[j]); len(lm) >= 2 {
				if rm := reRouteKV.FindStringSubmatch(lm[1]); len(rm) >= 2 {
					route = rm[1]
				}
			}
			j++
		}
		var a, b, c int
		if strings.Contains(state, "semacquire") {
			a = 1
		}
		if strings.Contains(state, "io wait") || strings.Contains(state, "syscall") {
			b = 1
		}
		if durSec >= 10 {
			if strings.Contains(state, "semacquire") || strings.Contains(state, "chan receive") || strings.Contains(state, "chan send") || strings.Contains(state, "select") || strings.Contains(state, "io wait") || strings.Contains(state, "syscall") {
				c = 1
			}
		}
		cur := stats[route]
		stats[route] = [3]int{cur[0] + a, cur[1] + b, cur[2] + c}
		i = j
	}
	return stats
}

type RouteStat struct {
	Route     string `json:"route"`
	Requests  int    `json:"requests"`
	BlockLock int    `json:"blockLock"`
	BlockIO   int    `json:"blockIO"`
	BlockPerm int    `json:"blockPerm"`
}

// RouteStats 合并最近10秒请求计数与按路由阻塞统计
func (t *Tracker) RouteStats() []RouteStat {
	reqs := t.requestsInWindowByRoute(10 * time.Second)
	blocks := classifyBlocksByRoute()
	routes := make(map[string]struct{})
	for r := range reqs {
		routes[r] = struct{}{}
	}
	for r := range blocks {
		routes[r] = struct{}{}
	}
	out := make([]RouteStat, 0, len(routes))
	for r := range routes {
		b := blocks[r]
		out = append(out, RouteStat{Route: r, Requests: reqs[r], BlockLock: b[0], BlockIO: b[1], BlockPerm: b[2]})
	}
	return out
}

// PushSample 将当前样本推入历史，最多保留 600 个点
func (t *Tracker) PushSample() {
	s := t.CurrentSample()
	t.histMu.Lock()
	t.history = append(t.history, s)
	if len(t.history) > 600 {
		t.history = t.history[len(t.history)-600:]
	}
	t.histMu.Unlock()
}

// History 返回历史拷贝
func (t *Tracker) History() []Sample {
	t.histMu.RLock()
	h := make([]Sample, len(t.history))
	copy(h, t.history)
	t.histMu.RUnlock()
	return h
}
