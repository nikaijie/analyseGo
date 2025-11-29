package metrics

import "sync"

type Hub struct {
	mu   sync.RWMutex
	subs []chan struct{}
}

func NewHub() *Hub { return &Hub{} }

// Subscribe 返回一个订阅通道，关闭连接时需调用 Unsubscribe
func (h *Hub) Subscribe() chan struct{} {
	ch := make(chan struct{}, 100)
	h.mu.Lock()
	h.subs = append(h.subs, ch)
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	for i, c := range h.subs {
		if c == ch {
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			break
		}
	}
	h.mu.Unlock()
}

// Notify 广播一次事件，唤醒所有订阅者
func (h *Hub) Notify() {
	h.mu.RLock()
	for _, ch := range h.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	h.mu.RUnlock()
}
