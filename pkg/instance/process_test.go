package instance

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestProcessState_StartDoesNotPanicOnProcessGroup is a regression test for
// the server 500 on instance start. Previously, setProcessGroup was called
// on cmd.Process.Pid BEFORE cmd.Start(), dereferencing a nil *os.Process and
// panicking inside the HTTP handler. With prepareProcessGroup setting
// SysProcAttr before Start, this must no longer panic.
func TestProcessState_StartDoesNotPanicOnProcessGroup(t *testing.T) {
	p := &processState{}
	var out, errOut bytes.Buffer

	// /bin/true exits immediately; we only care that start succeeds without
	// panicking and that the process group attribute was applied.
	if e := p.start("/bin/true", nil, nil, &out, &errOut); e != nil {
		t.Fatalf("start: %v", e)
	}

	if p.pid() == 0 {
		t.Fatal("pid is 0 after start")
	}
	if !p.isRunning() {
		// /bin/true may have exited already; that is acceptable, but the
		// process state must have recorded a non-zero pid during start.
		// Wait briefly for the process to finish.
		select {
		case <-p.done:
		case <-time.After(2 * time.Second):
			t.Fatal("process did not finish within 2s")
		}
	}

	// stop() must be safe to call after the process has exited.
	if err := p.stop(time.Second); err != nil {
		t.Fatalf("stop after exit: %v", err)
	}
}

// TestProcessState_StopBeforeStartIsNoOp ensures that calling stop() on a
// process that was never started does not panic.
func TestProcessState_StopBeforeStartIsNoOp(t *testing.T) {
	p := &processState{}
	if err := p.stop(time.Second); err != nil {
		t.Fatalf("stop on unstarted process: %v", err)
	}
	if p.isRunning() {
		t.Fatal("unstarted process reports running")
	}
	if p.pid() != 0 {
		t.Fatalf("unstarted process pid = %d, want 0", p.pid())
	}
}

// TestProcessState_StartCapturesOutput verifies that stdout/stderr from the
// spawned process are routed to the provided writers.
func TestProcessState_StartCapturesOutput(t *testing.T) {
	p := &processState{}
	var out, errOut bytes.Buffer

	if err := p.start("/bin/sh", []string{"-c", "echo hello-out; echo hello-err 1>&2"}, nil, &out, &errOut); err != nil {
		t.Fatalf("start: %v", err)
	}

	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
		t.Fatal("process did not finish within 2s")
	}

	if !strings.Contains(out.String(), "hello-out") {
		t.Errorf("stdout = %q, want it to contain hello-out", out.String())
	}
	if !strings.Contains(errOut.String(), "hello-err") {
		t.Errorf("stderr = %q, want it to contain hello-err", errOut.String())
	}
}
