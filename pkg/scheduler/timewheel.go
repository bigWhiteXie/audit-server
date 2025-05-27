package scheduler

import (
	"sync"
	"time"
)

type TimeWheel struct {
	ticker      *time.Ticker
	slots       []map[string]Task // 槽位使用Map去重
	currentSlot int
	taskQueue   *TaskQueue
	slotMutexes []sync.Mutex // 每个槽位独立锁
}

func NewTimeWheel(slotCount int, interval time.Duration, taskQueue *TaskQueue) *TimeWheel {
	tw := &TimeWheel{
		ticker:      time.NewTicker(interval),
		slots:       make([]map[string]Task, slotCount),
		slotMutexes: make([]sync.Mutex, slotCount),
		currentSlot: 0,
	}
	for i := range tw.slots {
		tw.slots[i] = make(map[string]Task)
	}
	go tw.Advance()
	return tw
}

func (tw *TimeWheel) AddTask(task Task) {
	now := time.Now()
	nextTime := now.Add(time.Duration(task.ExeInterval()) * time.Second)
	if nextTime.Before(now) {
		nextTime = now // 防止历史时间
	}

	duration := nextTime.Sub(now)
	slots := int(duration / (100 * time.Millisecond))
	targetSlot := (tw.currentSlot + slots) % len(tw.slots)

	tw.slotMutexes[targetSlot].Lock()
	defer tw.slotMutexes[targetSlot].Unlock()

	tw.slots[targetSlot][task.Name()] = task
}

func (tw *TimeWheel) Advance() {
	for range tw.ticker.C {
		tw.slotMutexes[tw.currentSlot].Lock()
		currentTasks := tw.slots[tw.currentSlot]
		tw.slots[tw.currentSlot] = make(map[string]Task) // 清空槽位
		tw.slotMutexes[tw.currentSlot].Unlock()

		for _, task := range currentTasks {
			if time.Now().After(task.NextRunTime()) {
				tw.taskQueue.Push(task)
			}
		}

		tw.currentSlot = (tw.currentSlot + 1) % len(tw.slots)
	}
}
