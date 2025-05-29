package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupTestDB(t *testing.T) *gorm.DB {
	datasource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		"root",
		"123456",
		"10.131.139.155",
		3306,
		"audit_log",
		"utf8",
		true,
		"Local")
	db, err := gorm.Open(mysql.Open(datasource))
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	taskEntry := &ScheduleTask{}
	db.Exec("drop table if exists " + taskEntry.TableName())
	if err := db.AutoMigrate(&ScheduleTask{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	taskEntry.TaskName = "test_task"
	taskEntry.NextRunTime = time.Now().Add(time.Duration(10) * time.Second)
	taskEntry.Priority = 1

	if err := db.Model(taskEntry).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_name"}},
		DoNothing: true,
	}).Create(taskEntry).Error; err != nil {
		panic(err)
	}
	return db
}

func TestMySQLLock_TryLockAndUnlock(t *testing.T) {
	db := setupTestDB(t)
	lock := NewMySQLLock(db, 15*time.Second)
	taskName := "test_task"

	taskEntry := &ScheduleTask{}
	taskEntry.TaskName = taskName
	taskEntry.NextRunTime = time.Now().Add(time.Duration(10) * time.Second)
	taskEntry.Priority = 1

	ok, err := lock.TryLock(context.Background(), taskName)
	if err != nil || !ok {
		t.Fatalf("TryLock failed: %v", err)
	}

	ok2, err2 := lock.TryLock(context.Background(), taskName)
	if err2 != nil || !ok2 {
		t.Fatalf("TryLock reentrant failed: %v", err2)
	}

	err3 := lock.Unlock(context.Background(), taskName)
	if err3 != nil {
		t.Fatalf("Unlock failed: %v", err3)
	}
}

func TestMySQLLock_AutoRenew(t *testing.T) {
	db := setupTestDB(t)
	db.Model(&ScheduleTask{}).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_name"}},
		DoNothing: true,
	}).Create(&ScheduleTask{
		TaskName:    "renew_task",
		NextRunTime: time.Now().Add(time.Duration(10) * time.Second),
		Priority:    1,
	})
	lock := NewMySQLLock(db, time.Millisecond*150)
	taskName := "renew_task"
	ok, err := lock.TryLock(context.Background(), taskName)
	if err != nil || !ok {
		t.Fatalf("TryLock failed: %v", err)
	}
	lock.AutoRenew()
	time.Sleep(time.Millisecond * 350)
	lock.ReleaseAll()
}
