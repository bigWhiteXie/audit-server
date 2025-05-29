package scheduler

import (
	"container/heap"
	"sync"
	"time"
)

// ScheduleEntry 表示调度表中的一项
// 可扩展字段如下次运行时间、锁状态、任务名等
// 具体实现可结合分布式锁与心跳机制
type ScheduleTask struct {
	Id             int       `gorm:"primaryKey;autoIncrement" json:"id"`                           // 任务ID
	TaskName       string    `gorm:"uniqueIndex:idx_task_name;type:varchar(155)" json:"task_name"` // 任务唯一标识，唯一索引
	LeaseHolder    string    `json:"lease_holder"`                                                 // 当前持有者（实例ID）
	LeaseUntil     time.Time `gorm:"default:null" json:"lease_until"`                              // 租约到期时间
	LastRunTime    time.Time `gorm:"default:null" json:"last_runtime"`                             // 上次执行时间
	NextRunTime    time.Time `gorm:"default:null" json:"next_runtime"`                             // 理论下次执行时间
	ExecutionCost  int64     `json:"execution_cost"`                                               // 平均耗时（ms）
	ExecutionCount int       `json:"execution_count"`                                              // 执行次数
	FailureCount   int       `json:"failure_count"`                                                // 连续失败次数
	Priority       int       `json:"priority"`                                                     // 优先级
}

func (t *ScheduleTask) TableName() string {
	return "schedule_task"
}

type TaskQueue struct {
	heap *taskHeap
	mu   sync.Mutex
	cond *sync.Cond
}

func NewTaskQueue() *TaskQueue {
	q := &TaskQueue{}
	q.cond = sync.NewCond(&q.mu)
	q.heap = NewTaskHeap()
	return q
}

// Push 将任务按优先级插入队列
func (q *TaskQueue) Push(task Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	heap.Push(q.heap, task)
	q.cond.Signal()
}

// Pop 阻塞直到队列非空，取出优先级最高且最早入队的任务
func (q *TaskQueue) Pop() Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	for q.heap.Len() == 0 {
		q.cond.Wait()
	}
	return heap.Pop(q.heap).(Task)
}

type taskHeap []Task

func NewTaskHeap() *taskHeap {
	h := &taskHeap{}
	heap.Init(h)
	return h
}

func (h taskHeap) Len() int { return len(h) }
func (h taskHeap) Less(i, j int) bool {
	// 优先级高的排前面，优先级相等时先进先出
	if (h[i]).Priority() == (h[j]).Priority() {
		return i < j
	}
	return (h[i]).Priority() > (h[j]).Priority()
}
func (h taskHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *taskHeap) Push(x interface{}) {
	*h = append(*h, x.(Task))
}
func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
