package instance

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type processState struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	pidValue    int
	stdout      io.Writer
	stderr      io.Writer
	done        chan struct{}
}

func (p *processState) start(binaryPath string, args []string, env []string, stdout, stderr io.Writer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := prepareProcessGroup(cmd); err != nil {
		return fmt.Errorf("prepare process group: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	p.cmd = cmd
	p.pidValue = cmd.Process.Pid
	p.stdout = stdout
	p.stderr = stderr
	p.done = make(chan struct{})

	go p.wait()
	return nil
}

func (p *processState) wait() {
	defer close(p.done)
	if err := p.cmd.Wait(); err != nil {
		// Log the error but don't panic
	}
}

func (p *processState) stop(gracePeriod time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	if err := killProcessGroup(p.pidValue); err != nil {
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill process: %w", err)
		}
		return nil
	}

	select {
	case <-p.done:
		return nil
	case <-time.After(gracePeriod):
		if err := killProcessGroup(p.pidValue); err != nil {
			return p.cmd.Process.Kill()
		}
		<-p.done
		return nil
	}
}

func (p *processState) isRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cmd != nil && p.cmd.Process != nil
}

func (p *processState) pid() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pidValue
}
