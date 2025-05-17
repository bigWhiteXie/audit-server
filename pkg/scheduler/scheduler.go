package scheduler

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScheduleConfig struct {
	FailThreshold   int `json:"" yaml:"FailThreshold"`   // 失败阈值，达到后开启熔断
	IsolateDuration int `json:"" yaml:"IsolateDuration"` // 熔断时间，单位秒
	LeaseDuration   int `json:"" yaml:"LeaseDuration"`   // 锁租期，单位秒
}

type Scheduler struct {
	ScheduleConfig

	db             *gorm.DB
	taskQueue      *TaskQueue
	timeWheel      *TimeWheel
	cancelFunc     context.CancelFunc
	lock           DistributedLock
	circuitBreaker CircuitBreaker
}

func NewScheduler(db *gorm.DB, config ScheduleConfig) *Scheduler {
	taskQueue := NewTaskQueue()

	return &Scheduler{
		db:             db,
		taskQueue:      taskQueue,
		timeWheel:      NewTimeWheel(time.Millisecond*50, taskQueue),
		lock:           NewMySQLLock(db, time.Duration(config.LeaseDuration)*time.Second),
		circuitBreaker: *NewCircuitBreaker(config.FailThreshold, time.Duration(config.IsolateDuration)*time.Second),
	}
}

func (s *Scheduler) RegisterTask(task Task) {
	// 先查询数据库中是否存在该任务，若存在则直接读取
	taskEntry := &ScheduleTask{}
	if err := s.db.Model(taskEntry).Where("task_name = ?", task.Name()).First(taskEntry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			taskEntry.TaskName = task.Name()
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

func (s *Scheduler) Start() {
	ctx, cancelFunc := context.WithCancel(context.TODO())
	s.cancelFunc = cancelFunc

	s.timeWheel.Run()
	s.lock.AutoRenew()
	for {
		select {
		case <-ctx.Done():
			logx.Info("=========================scheduler stopped by ctx Done=========================")
			return
		default:
			task := s.taskQueue.Pop()
			// 尝试获取锁
			if ok, err := s.lock.TryLock(ctx, task.Name()); !ok || err != nil {
				logx.Info("=========================scheduler get lock failed=========================")
				if err != nil {
					logx.Errorf("get lock failed, err: %v", err)
				}
				task.SetNextRunTime(time.Now().Add(time.Duration(task.ExeInterval()) * time.Second))
				s.timeWheel.AddTask(task)
				continue
			}

			if s.circuitBreaker.IsIsolated(task.Name()) {
				logx.Infof("task is isolated, taskId: %s", task.Name())
				task.SetNextRunTime(time.Now().Add(time.Duration(task.ExeInterval()) * time.Second))
				s.timeWheel.AddTask(task)
				continue
			}

			go s.runTask(task)
		}
	}
}

func (s *Scheduler) Stop() {
	// 停止执行任务的协程
	s.cancelFunc()
	// 释放所有分布式锁
	s.lock.ReleaseAll()
	// 停止时间轮
	s.timeWheel.Stop()
}

func (s *Scheduler) runTask(task Task) {
	taskEntry := &ScheduleTask{}
	defer func() {
		if err := recover(); err != nil {
			logx.Errorf("task run failed, taskId: %s, err: %v", task.Name(), err)
		}

		nextTime := time.Now().Add(time.Duration(task.ExeInterval()) * time.Second)
		task.SetNextRunTime(nextTime)
		taskEntry.NextRunTime = nextTime

		// 更新任务信息并重新到时间轮
		s.timeWheel.AddTask(task)
		if err := s.db.Model(taskEntry).Where("task_name = ?", task.Name()).Updates(taskEntry).Error; err != nil {
			logx.Errorf("update taskEntry failed, taskName: %s, err: %v", task.Name(), err)
		}
	}()

	// 查询任务信息
	if err := s.db.Model(taskEntry).Where("task_name = ?", task.Name()).First(taskEntry).Error; err != nil {
		logx.Errorf("find taskEntry failed, taskName: %s", task.Name())
		return
	}

	// 调整到下次执行时间
	if time.Now().Before(taskEntry.NextRunTime) {
		time.Sleep(taskEntry.NextRunTime.Sub(time.Now()))
	}

	taskEntry.LastRunTime = time.Now()
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
	if s.circuitBreaker.IsIsolated(taskEntry.TaskName) {
		s.lock.Unlock(context.TODO(), taskEntry.TaskName)
	}
}

func (s *Scheduler) onSuccess(taskEntry *ScheduleTask) {
	s.circuitBreaker.OnSuccess(taskEntry.TaskName)
	taskEntry.FailureCount = 0
}
