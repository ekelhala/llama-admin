package instance

import (
	"fmt"
	"sync"
)

type Status string

const (
	StatusStopped       Status = "stopped"
	StatusRunning       Status = "running"
	StatusFailed        Status = "failed"
	StatusRestarting    Status = "restarting"
	StatusShuttingDown  Status = "shutting_down"
)

type statusState struct {
	mu             sync.Mutex
	status         Status
	onStatusChange func(Status)
}

func newStatusState(onChange func(Status)) *statusState {
	return &statusState{status: StatusStopped, onStatusChange: onChange}
}

func (s *statusState) Get() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *statusState) Set(newStatus Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	allowed := statusTransitions[s.status]
	if allowed == nil {
		return fmt.Errorf("no transitions from status %q", s.status)
	}
	if !allowed[newStatus] {
		return fmt.Errorf("invalid transition: %s -> %s", s.status, newStatus)
	}

	s.status = newStatus
	if s.onStatusChange != nil {
		s.onStatusChange(newStatus)
	}
	return nil
}

var statusTransitions = map[Status]map[Status]bool{
	StatusStopped: {
		StatusRestarting: true,
	},
	StatusRestarting: {
		StatusRunning:    true,
		StatusFailed:     true,
		StatusStopped:    true,
	},
	StatusRunning: {
		StatusShuttingDown: true,
		StatusFailed:       true,
	},
	StatusShuttingDown: {
		StatusStopped: true,
	},
	StatusFailed: {
		StatusRestarting: true,
	},
}
