package scheduler

import (
	"sync"
	"time"
)

// CircuitBreaker 熔断器，按任务名隔离
// 支持连续失败N次自动熔断，隔离一段时间后自动恢复
type CircuitBreaker struct {
	failCount   map[string]int
	trippedAt   map[string]time.Time
	threshold   int
	isolateTime time.Duration
	mu          sync.Mutex
}

func NewCircuitBreaker(threshold int, isolateTime time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failCount:   make(map[string]int),
		trippedAt:   make(map[string]time.Time),
		threshold:   threshold,
		isolateTime: isolateTime,
	}
}

func (cb *CircuitBreaker) OnSuccess(task string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failCount[task] = 0
	delete(cb.trippedAt, task)
}

func (cb *CircuitBreaker) OnFailure(task string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failCount[task]++
	if cb.failCount[task] >= cb.threshold {
		cb.trippedAt[task] = time.Now()
	}
}

func (cb *CircuitBreaker) IsIsolated(task string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	tripped, ok := cb.trippedAt[task]
	if !ok {
		return false
	}
	if time.Since(tripped) > cb.isolateTime {
		cb.failCount[task] = 0
		delete(cb.trippedAt, task)
		return false
	}
	return true
}
