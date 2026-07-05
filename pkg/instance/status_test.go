package instance

import (
	"testing"
)

func TestStatusState_InitialStatusIsStopped(t *testing.T) {
	s := newStatusState(nil)
	if got := s.Get(); got != StatusStopped {
		t.Fatalf("initial status = %q, want stopped", got)
	}
}

func TestStatusState_ValidTransitions(t *testing.T) {
	cases := []struct {
		from, to Status
	}{
		{StatusStopped, StatusRestarting},
		{StatusRestarting, StatusRunning},
		{StatusRestarting, StatusFailed},
		{StatusRestarting, StatusStopped},
		{StatusRunning, StatusShuttingDown},
		{StatusRunning, StatusFailed},
		{StatusShuttingDown, StatusStopped},
		{StatusFailed, StatusRestarting},
	}
	for _, c := range cases {
		s := newStatusState(nil)
		// Force the starting state without going through transitions.
		s.status = c.from
		if err := s.Set(c.to); err != nil {
			t.Errorf("expected valid transition %s -> %s, got error: %v", c.from, c.to, err)
		}
		if got := s.Get(); got != c.to {
			t.Errorf("after transition %s -> %s, status = %q", c.from, c.to, got)
		}
	}
}

func TestStatusState_InvalidTransitions(t *testing.T) {
	cases := []struct {
		from, to Status
	}{
		{StatusStopped, StatusRunning},      // must go through restarting
		{StatusStopped, StatusShuttingDown}, // not running
		{StatusRunning, StatusStopped},      // must go through shutting_down
		{StatusRunning, StatusRestarting},   // must stop first
		{StatusShuttingDown, StatusRunning}, // only -> stopped
		{StatusFailed, StatusRunning},       // must go through restarting
	}
	for _, c := range cases {
		s := newStatusState(nil)
		s.status = c.from
		if err := s.Set(c.to); err == nil {
			t.Errorf("expected error for invalid transition %s -> %s, got nil", c.from, c.to)
		}
		if got := s.Get(); got != c.from {
			t.Errorf("status changed after invalid transition %s -> %s: now %q", c.from, c.to, got)
		}
	}
}

func TestStatusState_OnChangeInvoked(t *testing.T) {
	var observed []Status
	s := newStatusState(func(newStatus Status) {
		observed = append(observed, newStatus)
	})
	s.status = StatusStopped

	if err := s.Set(StatusRestarting); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(StatusRunning); err != nil {
		t.Fatal(err)
	}

	if len(observed) != 2 || observed[0] != StatusRestarting || observed[1] != StatusRunning {
		t.Fatalf("onChange observed %v, want [restarting running]", observed)
	}
}

func TestStatusState_StopFromStoppedIsNoOp(t *testing.T) {
	// The instance.Stop() early-returns when already stopped; the state
	// machine itself has no stopped->stopped transition, mirroring that.
	s := newStatusState(nil)
	if err := s.Set(StatusStopped); err == nil {
		t.Error("expected error for stopped -> stopped")
	}
}
