package scheduler

import (
	"time"

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
	taskMap        map[string]Task
	lock           DistributedLock
	circuitBreaker CircuitBreaker
	minInterval    int64 // 最小执行间隔，单位秒
}

func NewScheduler(db *gorm.DB, lock DistributedLock, config ScheduleConfig) *Scheduler {
	return &Scheduler{
		db:      db,
		taskMap: make(map[string]Task),
		lock:    lock,
		circuitBreaker: CircuitBreaker{
			failCount:   make(map[string]int),
			trippedAt:   make(map[string]time.Time),
			threshold:   config.FailThreshold,
			isolateTime: time.Duration(config.IsolateDuration) * time.Second,
		},
	}
}

func (s *Scheduler) RegisterTask(task Task) {
	s.taskMap[task.Name()] = task

	// 尝试往数据库中插入任务，若已存在则忽略
	taskEntry := &ScheduleTask{
		TaskID:      task.Name(),
		NextRunTime: time.Now().Add(time.Duration(task.ExeInterval()) * time.Second),
		Priority:    task.Priority(),
	}
	if s.minInterval == 0 || s.minInterval > task.ExeInterval() {
		s.minInterval = task.ExeInterval()
	}
	if err := s.db.Model(taskEntry).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_id"}},
		DoNothing: true,
	}).Create(taskEntry).Error; err != nil {
		panic(err)
	}
}

// func (s *Scheduler) Start(ctx context.Context) {
// 	ticker := time.NewTicker(time.Duration(s.minInterval) * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			logx.Info("=========================scheduler stopped by ctx Done=========================")
// 			return
// 		case <-ticker.C:
// 			s.runTasks()
// 		}
// 	}
// }
