package scheduler

import (
	"context"
	"sync"
	"time"

	"codexie.com/auditlog/pkg/util"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// LockStatus 锁状态
const (
	LockFree = 0 // 未被占用
	LockHeld = 1 // 已被占用
)

// 负责对任务上锁、解锁、自动续期
type MySQLLock struct {
	db            *gorm.DB
	instanceName  string
	leaseDuration time.Duration // 锁的租约时间
	lockMap       sync.Map      // 记录上次锁续期的时间
	ticker        *time.Ticker
}

func NewMySQLLock(db *gorm.DB, leaseDuration time.Duration) *MySQLLock {
	ip, err := util.GetLocalIP()
	if err != nil {
		panic(err)
	}
	instanceName := ip + "-" + uuid.New().String()[:12]

	lock := &MySQLLock{
		db:            db,
		leaseDuration: leaseDuration,
		lockMap:       sync.Map{},
		instanceName:  instanceName,
	}

	return lock
}

func (l *MySQLLock) TryLock(ctx context.Context, taskName string) (bool, error) {
	now := time.Now()
	leaseTime := l.leaseDuration

	// ===========================判断当前是否持有锁且锁未过期===========================
	if lockTime, ok := l.lockMap.Load(taskName); ok {
		lockTime, _ := lockTime.(time.Time)
		if now.Sub(lockTime) < leaseTime {
			return true, nil
		}
	}

	// ===========================说明当前不持有锁、尝试从数据库获取锁并更新到map===========================
	task := &ScheduleTask{
		LeaseHolder: l.instanceName,
		LeaseUntil:  now.Add(leaseTime),
	}
	// 只更新task中非零值字段(数据库操作是并发安全的)
	res := l.db.WithContext(ctx).Model(&ScheduleTask{}).
		Where("task_name = ? AND (lease_holder = '' OR lease_until < ?)", taskName, now).
		Updates(task)
	if res.Error != nil {
		logx.Errorf("fail to get task lock,cause:%s", res.Error)
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		logx.Debugf("fail to get task lock,taskName:%s", taskName)
		l.lockMap.Delete(taskName)
		return false, nil
	}

	// 获取锁成功
	l.lockMap.Store(taskName, now)
	return true, nil
}

func (l *MySQLLock) Unlock(ctx context.Context, taskName string) error {
	// 尝试获取锁,只更新task中非零值字段
	res := l.db.WithContext(ctx).
		Model(&ScheduleTask{}).
		Select("lease_holder", "lease_until").
		Where("task_name = ? AND lease_holder = ? and lease_until > ?", taskName, l.instanceName, time.Now()).
		Updates(&ScheduleTask{})
	if res.Error != nil {
		logx.Errorf("fail to get task lock,cause:%s", res.Error)
		return res.Error
	}

	//说明锁已经被释放
	if res.RowsAffected == 0 {
		logx.Alert("lock already released")
	}
	l.lockMap.Delete(taskName)

	return nil
}

func (l *MySQLLock) ReleaseAll() {
	l.ticker.Stop()
	wg := sync.WaitGroup{}
	l.lockMap.Range(func(key, value any) bool {
		taskName, _ := key.(string)
		wg.Add(1)
		go func(taskName string) {
			defer wg.Done()
			err := l.Unlock(context.Background(), taskName)
			if err != nil {
				logx.Errorf("fail to release task lock,cause:%s", err)
			}
		}(taskName)
		return true
	})

	wg.Wait()
}

func (l *MySQLLock) AutoRenew() {
	l.ticker = time.NewTicker(l.leaseDuration / 3)
	go func() {
		for range l.ticker.C {
			l.renewLocks()
		}
	}()
}

func (l *MySQLLock) renewLocks() {
	now := time.Now()
	wg := sync.WaitGroup{}
	l.lockMap.Range(func(key, value any) bool {
		taskName, _ := key.(string)
		lockTime, _ := value.(time.Time)
		if now.Sub(lockTime) < l.leaseDuration {
			wg.Add(1)
			go func(taskName string) {
				defer wg.Done()
				// 锁未过期，尝试续期
				leaseTime := now.Add(l.leaseDuration)
				res := l.db.Model(&ScheduleTask{}).Where("task_name =? AND lease_holder =? AND lease_until >?", taskName, l.instanceName, now).Update("lease_until", leaseTime)
				if res.Error != nil {
					logx.Errorf("fail to renew task lock,cause:%s", res.Error)
					l.lockMap.Delete(taskName)
					return
				}
				if res.RowsAffected == 0 {
					logx.Errorf("lock of %s already released", taskName)
					l.lockMap.Delete(taskName)
				}
				l.lockMap.Store(taskName, now)
			}(taskName)
		}
		return true
	})

	wg.Wait()
}
