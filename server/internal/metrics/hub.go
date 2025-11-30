package metrics

import "sync"

// Hub 实现发布订阅模式，用于通知 SSE 客户端有新数据更新
type Hub struct {
	mu   sync.RWMutex
	subs []chan struct{} // 订阅者通道列表
}

// NewHub 创建新的 Hub 实例
func NewHub() *Hub {
	return &Hub{
		subs: make([]chan struct{}, 0),
	}
}

// Subscribe 订阅通知，返回一个通知通道
// 注意：使用完毕后必须调用 Unsubscribe 以避免内存泄漏
func (h *Hub) Subscribe() chan struct{} {
	ch := make(chan struct{}, 100) // 带缓冲的通道，避免阻塞
	h.mu.Lock()
	h.subs = append(h.subs, ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe 取消订阅，移除指定的通知通道
func (h *Hub) Unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i, c := range h.subs {
		if c == ch {
			// 移除通道
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			close(ch) // 关闭通道
			break
		}
	}
}

// Notify 广播通知，唤醒所有订阅者
// 使用非阻塞方式发送，避免订阅者未就绪时阻塞
func (h *Hub) Notify() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subs {
		select {
		case ch <- struct{}{}:
			// 成功发送
		default:
			// 通道已满，跳过（避免阻塞）
		}
	}
}
