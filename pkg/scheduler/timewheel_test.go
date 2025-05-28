package scheduler

import (
	"testing"
	"time"
)

type mockTask struct {
	name        string
	nextRunTime time.Time
	priority    int
	runCalled   *bool
}

func (m *mockTask) Name() string               { return m.name }
func (m *mockTask) NextRunTime() time.Time     { return m.nextRunTime }
func (m *mockTask) SetNextRunTime(t time.Time) { m.nextRunTime = t }
func (m *mockTask) Priority() int              { return m.priority }
func (m *mockTask) ExeInterval() int64         { return 1 }
func (m *mockTask) Run() error                 { *m.runCalled = true; return nil }

func TestTimeWheel_AddTaskAndRun(t *testing.T) {
	queue := NewTaskQueue()
	tw := NewTimeWheel(10, time.Millisecond*50, queue)
	runCalled := false
	task := &mockTask{
		name:        "test",
		nextRunTime: time.Now().Add(time.Millisecond * 60),
		priority:    1,
		runCalled:   &runCalled,
	}
	tw.taskQueue = queue
	tw.AddTask(task)
	tw.Run()
	time.Sleep(time.Millisecond * 120)
	popTask := queue.Pop()
	if popTask.Name() != "test" {
		t.Errorf("expected task name 'test', got %s", popTask.Name())
	}
	tw.Stop()
}

func TestTimeWheel_Stop(t *testing.T) {
	queue := NewTaskQueue()
	tw := NewTimeWheel(5, time.Millisecond*20, queue)
	tw.Run()
	tw.Stop()
	// 多次Stop应无panic
	tw.Stop()
}
