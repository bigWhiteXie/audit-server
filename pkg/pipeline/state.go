package pipeline

import "sync/atomic"

type State struct {
	status atomic.Int32
}

const (
	StatusNormal = iota
	StatusBlocked
	StatusRecovering
)

func NewState() *State {
	s := &State{}
	s.status.Store(StatusNormal)
	return s
}

func (s *State) IsBlocked() bool {
	return s.status.Load() == StatusBlocked
}

func (s *State) EnterBlocked() {
	s.status.Store(StatusBlocked)
}

func (s *State) EnterRecovering() {
	s.status.Store(StatusRecovering)
}

func (s *State) EnterNormal() {
	s.status.Store(StatusNormal)
}

func (s *State) GetStatus() int32 {
	return s.status.Load()
}
