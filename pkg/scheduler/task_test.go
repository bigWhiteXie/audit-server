package scheduler

import (
	"sync"
	"testing"
	"time"
)

type simpleTask struct {
	name     string
	priority int
}

func (s *simpleTask) Name() string               { return s.name }
func (s *simpleTask) NextRunTime() time.Time     { return time.Now() }
func (s *simpleTask) SetNextRunTime(t time.Time) {}
func (s *simpleTask) Priority() int              { return s.priority }
func (s *simpleTask) ExeInterval() int64         { return 1 }
func (s *simpleTask) Run() error                 { return nil }

func TestTaskQueue_PushAndPop(t *testing.T) {
	q := NewTaskQueue()
	t1 := &simpleTask{name: "t1", priority: 1}
	t2 := &simpleTask{name: "t2", priority: 3}
	t3 := &simpleTask{name: "t3", priority: 2}

	q.Push(t1)
	q.Push(t2)
	q.Push(t3)

	pop1 := q.Pop()
	if pop1.Name() != "t2" {
		t.Errorf("expected t2, got %s", pop1.Name())
	}
	pop2 := q.Pop()
	if pop2.Name() != "t3" {
		t.Errorf("expected t3, got %s", pop2.Name())
	}
	pop3 := q.Pop()
	if pop3.Name() != "t1" {
		t.Errorf("expected t1, got %s", pop3.Name())
	}
}

func TestTaskQueue_BlockingPop(t *testing.T) {
	q := NewTaskQueue()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		pop := q.Pop()
		if pop.Name() != "block" {
			t.Errorf("expected block, got %s", pop.Name())
		}
	}()
	time.Sleep(time.Millisecond * 50)
	q.Push(&simpleTask{name: "block", priority: 1})
	wg.Wait()
}
