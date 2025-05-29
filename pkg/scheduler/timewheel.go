package scheduler

import (
	"sync"
	"time"
)

type TimeWheel struct {
	ticker      *time.Ticker
	slots       []map[string]Task
	currentSlot int
	taskQueue   *TaskQueue
	slotMutexes []sync.Mutex
}

func NewTimeWheel(interval time.Duration, taskQueue *TaskQueue) *TimeWheel {
	slotCount := int(500 / interval.Milliseconds())
	tw := &TimeWheel{
		ticker:      time.NewTicker(interval),
		taskQueue:   taskQueue,
		slots:       make([]map[string]Task, slotCount),
		slotMutexes: make([]sync.Mutex, slotCount),
		currentSlot: 0,
	}
	for i := range tw.slots {
		tw.slots[i] = make(map[string]Task)
	}

	return tw
}

// 添加任务，任务会根据下次执行时间来分配到对应槽位
func (tw *TimeWheel) AddTask(task Task) {
	now := time.Now()
	nextTime := task.NextRunTime()
	if nextTime.Before(now) {
		nextTime = now // 防止历史时间
	}

	duration := nextTime.Sub(now)
	spanSlots := int(duration / (100 * time.Millisecond))
	targetSlot := (tw.currentSlot + spanSlots + 1) % len(tw.slots)

	tw.slotMutexes[targetSlot].Lock()
	defer tw.slotMutexes[targetSlot].Unlock()

	tw.slots[targetSlot][task.Name()] = task
}

func (tw *TimeWheel) Run() {
	go func() {
		for range tw.ticker.C {
			tw.slotMutexes[tw.currentSlot].Lock()
			currentTasks := tw.slots[tw.currentSlot]
			tw.slots[tw.currentSlot] = make(map[string]Task) // 清空槽位
			tw.slotMutexes[tw.currentSlot].Unlock()

			for _, task := range currentTasks {
				if time.Now().After(task.NextRunTime()) {
					tw.taskQueue.Push(task)
				} else {
					tw.AddTask(task)
				}
			}

			tw.currentSlot = (tw.currentSlot + 1) % len(tw.slots)
		}
	}()
}

func (tw *TimeWheel) Stop() {
	tw.ticker.Stop()
}
