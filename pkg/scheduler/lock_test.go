package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	datasource := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		"root",
		"j3391111",
		"192.168.126.100",
		3306,
		"audit_log",
		"utf8",
		true,
		"Local")
	db, err := gorm.Open(mysql.Open(datasource))
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&ScheduleTask{})
	return db
}

func TestMySQLLock_TryLockAndUnlock(t *testing.T) {
	db := setupTestDB(t)
	lock := NewMySQLLock(db, time.Second)
	taskName := "test_task"

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
	lock := NewMySQLLock(db, time.Millisecond*100)
	taskName := "renew_task"
	ok, err := lock.TryLock(context.Background(), taskName)
	if err != nil || !ok {
		t.Fatalf("TryLock failed: %v", err)
	}
	lock.AutoRenew()
	time.Sleep(time.Millisecond * 350)
	lock.ReleaseAll()
}
