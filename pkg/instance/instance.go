package instance

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"llama-admin/pkg/config"
)

type Instance struct {
	ID          int64
	Name        string
	RawStatus   Status
	CreatedAt   int64
	UpdatedAt   int64
	Opts        *Options
	Host        string
	Port        int
	PID         int
	OwnerUserID *int64

	mu         sync.Mutex
	status     *statusState
	process    *processState
	proxy      *proxyState
	logger     *Logger
	cfg        *config.AppConfig
}

func New(id int64, name string, opts *Options, host string, port int, cfg *config.AppConfig) (*Instance, error) {
	if err := opts.ValidateAndApplyDefaults(); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}

	if host == "" {
		host = "127.0.0.1"
	}

	now := time.Now().Unix()
	inst := &Instance{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
		Opts:      opts,
		Host:      host,
		Port:      port,
		cfg:       cfg,
	}

	inst.status = newStatusState(func(newStatus Status) {
		inst.UpdatedAt = time.Now().Unix()
	})

	inst.proxy = newProxyState(host, port)

	log, err := NewLogger(name, "")
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	inst.logger = log

	inst.process = &processState{}

	return inst, nil
}

func (i *Instance) Start() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.status.Set(StatusRestarting) != nil {
		// Status might already be running or transitioning
		current := i.status.Get()
		if current == StatusRunning {
			return fmt.Errorf("instance %s is already running", i.Name)
		}
	}

	// Build command args
	args := []string{}
	if model, ok := i.Opts.BackendOptions["model"]; ok {
		if m, ok := model.(string); ok && m != "" {
			args = append(args, fmt.Sprintf("--model=%s", m))
		}
	}
	args = append(args, fmt.Sprintf("--host=%s", i.Host))
	if i.Port != 0 {
		args = append(args, fmt.Sprintf("--port=%d", i.Port))
	}
	if ctxSize, ok := i.Opts.BackendOptions["ctx_size"]; ok {
		if c, ok := ctxSize.(int); ok && c != 0 {
			args = append(args, fmt.Sprintf("--ctx-size=%d", c))
		}
	}
	if ngpu, ok := i.Opts.BackendOptions["n_gpu_layers"]; ok {
		if n, ok := ngpu.(int); ok && n != 0 {
			args = append(args, fmt.Sprintf("--n-gpu-layers=%d", n))
		}
	}

	// Build environment
	binaryPath := i.cfg.Backends.LlamaCpp.BinaryPath
	if binaryPath == "" {
		return fmt.Errorf("llama-server binary path not configured")
	}

	env := append(os.Environ(), "LLAMA_CACHE="+i.cfg.Backends.LlamaCpp.CacheDir)
	for k, v := range i.Opts.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := i.process.start(binaryPath, args, env, i.logger, i.logger); err != nil {
		i.status.Set(StatusFailed)
		return fmt.Errorf("start process: %w", err)
	}

	// Wait for healthy
	ctx, cancel := context.WithTimeout(context.Background(), i.cfg.Instances.OnDemandStartTimeout)
	defer cancel()

	if err := i.proxy.WaitForHealthy(ctx, i.Host, i.Port, i.cfg.Instances.OnDemandStartTimeout); err != nil {
		i.process.stop(5 * time.Second)
		i.status.Set(StatusFailed)
		return fmt.Errorf("wait for healthy: %w", err)
	}

	i.proxy.markHealthy()
	i.status.Set(StatusRunning)

	return nil
}

func (i *Instance) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.status.Get() == StatusStopped {
		return nil
	}

	if err := i.status.Set(StatusShuttingDown); err != nil {
		return err
	}

	i.proxy.setShuttingDown()

	if err := i.process.stop(10 * time.Second); err != nil {
		return fmt.Errorf("stop process: %w", err)
	}

	i.PID = 0
	i.status.Set(StatusStopped)
	return nil
}

func (i *Instance) Restart() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if err := i.Stop(); err != nil {
		return err
	}
	return i.Start()
}

func (i *Instance) WaitForHealthy(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return i.proxy.WaitForHealthy(ctx, i.Host, i.Port, timeout)
}

func (i *Instance) Logs(lines int) (string, error) {
	if i.logger == nil {
		return "", nil
	}
	return "", fmt.Errorf("logs not yet implemented")
}

func (i *Instance) Status() Status {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.RawStatus
}

func (i *Instance) SetStatus(s Status) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.RawStatus = s
}

func (i *Instance) MarkRunning(port int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Port = port
	i.RawStatus = StatusRunning
	i.proxy.markHealthy()
}

func (i *Instance) MarkStopped() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.PID = 0
	i.RawStatus = StatusStopped
}

func (i *Instance) ShouldTimeout() bool {
	return false
}

func (i *Instance) GetInflightRequests() int64 {
	return i.proxy.inflight.Load()
}

func (i *Instance) Proxy() *proxyState {
	return i.proxy
}
