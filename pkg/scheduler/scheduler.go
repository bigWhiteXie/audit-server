package scheduler

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScheduleConfig struct {
	FailThreshold   int // 失败阈值，达到后开启熔断
	IsolateDuration int // 熔断时间，单位秒
	LeaseDuration   int // 锁租期，单位秒
}

type Scheduler struct {
	ScheduleConfig

	db             *gorm.DB
	taskQueue      *TaskQueue
	timeWheel      *TimeWheel
	lock           DistributedLock
	circuitBreaker CircuitBreaker
}

func NewScheduler(db *gorm.DB, lock DistributedLock, config ScheduleConfig) *Scheduler {
	taskQueue := NewTaskQueue()

	return &Scheduler{
		db:             db,
		taskQueue:      taskQueue,
		timeWheel:      NewTimeWheel(100, time.Millisecond*100, taskQueue),
		lock:           lock,
		circuitBreaker: *NewCircuitBreaker(config.FailThreshold, time.Duration(config.IsolateDuration)*time.Second),
	}
}

func (s *Scheduler) RegisterTask(task Task) {
	// 先查询数据库中是否存在该任务，若存在则直接读取
	taskEntry := &ScheduleTask{}
	if err := s.db.Model(taskEntry).Where("task_name = ?", task.Name()).First(task.Name()).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			taskEntry.NextRunTime = time.Now().Add(time.Duration(task.ExeInterval()) * time.Second)
			taskEntry.Priority = task.Priority()

			if err := s.db.Model(taskEntry).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "task_name"}},
				DoNothing: true,
			}).Create(taskEntry).Error; err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	task.SetNextRunTime(taskEntry.NextRunTime)
	s.timeWheel.AddTask(task)
}

func (s *Scheduler) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logx.Info("=========================scheduler stopped by ctx Done=========================")
			return
		default:
			task := s.taskQueue.Pop()
			s.runTask(task)
		}
	}
}

func (s *Scheduler) runTask(task Task) {
	taskEntry := &ScheduleTask{}
	isRun := false
	defer func() {
		if err := recover(); err != nil {
			logx.Error("task run failed, taskId: %s, err: %v", task.Name(), err)
		}
		task.SetNextRunTime(time.Now().Add(time.Duration(task.ExeInterval()) * time.Second))
		s.timeWheel.AddTask(task)
		if isRun {
			if err := s.db.Model(taskEntry).Where("task_name = ?", task.Name()).Updates(taskEntry).Error; err != nil {
				logx.Error("update taskEntry failed, taskName: %s, err: %v", task.Name(), err)
			}
		}
	}()

	if s.circuitBreaker.IsIsolated(task.Name()) {
		logx.Infof("task is isolated, taskId: %s", task.Name())
		return
	}

	// 查询任务信息
	taskEntry.LastRunTime = time.Now()
	if err := s.db.Model(taskEntry).Where("task_name = ?", task.Name()).First(taskEntry).Error; err != nil {
		logx.Error("find taskEntry failed, taskName: %s", task.Name())
		return
	}

	// 调整到下次执行时间
	if time.Now().After(taskEntry.NextRunTime) {
		time.Sleep(taskEntry.NextRunTime.Sub(time.Now()))
	}
	if err := task.Run(); err != nil {
		s.onFailure(taskEntry, err)
		return
	}

	// 更新任务、熔断器状态等
	s.onSuccess(taskEntry)

}

func (s *Scheduler) onFailure(taskEntry *ScheduleTask, err error) {
	s.circuitBreaker.OnFailure(taskEntry.TaskName)
	taskEntry.FailureCount++
	logx.Error("task run failed, taskId: %s, err: %v", taskEntry.TaskName, err)
}

func (s *Scheduler) onSuccess(taskEntry *ScheduleTask) {
	s.circuitBreaker.OnSuccess(taskEntry.TaskName)
	taskEntry.FailureCount = 0
}
