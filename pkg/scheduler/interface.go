package scheduler

import (
	"context"
	"time"
)

// Task 定义任务接口，业务方需实现该接口
// Run方法为任务执行入口，返回是否成功和错误信息
// Name方法返回任务名称
// NextRunTime返回下次调度时间（如cron表达式解析结果）
type Task interface {
	Name() string
	Run() error
	ExeInterval() int64 // 执行间隔，单位秒
	NextRunTime() time.Time
	Priority() int
}

// DistributedLock 分布式锁接口，便于后续扩展不同存储实现（如Redis、DB等）
type DistributedLock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error) // 尝试加锁，返回是否成功
	Unlock(ctx context.Context, key string) error                             // 释放锁
	Renew(ctx context.Context, key string, ttl time.Duration) error           // 续期锁
	IsLocked(ctx context.Context, key string) (bool, error)                   // 判断锁是否存在
}
