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

	histMu  sync.RWMutex
	history []Sample
}

func NewTracker() *Tracker {
	return &Tracker{}
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
