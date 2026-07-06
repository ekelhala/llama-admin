package manager

import (
	"fmt"
	"log"
	"time"

	"llama-admin/pkg/instance"
)

func (m *manager) CreateInstance(name string, opts *instance.Options) (*instance.Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.registry.Get(name); exists {
		return nil, fmt.Errorf("instance %q already exists", name)
	}

	// Allocate port
	port, err := m.portAllocator.Allocate(name, 0)
	if err != nil {
		return nil, fmt.Errorf("allocate port: %w", err)
	}

	resolver := func(alias string) (string, error) {
		return m.modelMgr.ResolveAlias(alias)
	}
	inst, err := instance.New(0, name, opts, "127.0.0.1", port, m.cfg, resolver)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}

	m.registry.Add(inst)

	// Save to DB
	if err := m.instanceStore.Save(inst); err != nil {
		m.registry.Delete(name)
		m.portAllocator.Free(port)
		return nil, fmt.Errorf("save instance: %w", err)
	}

	return inst, nil
}

func (m *manager) GetInstance(name string) (*instance.Instance, error) {
	inst, ok := m.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("instance %q not found", name)
	}
	return inst, nil
}

func (m *manager) ListInstances() ([]*instance.Instance, error) {
	return m.registry.List(), nil
}

func (m *manager) StartInstance(name string) (*instance.Instance, error) {
	m.mu.Lock()
	inst, ok := m.registry.Get(name)
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("instance %q not found", name)
	}

	if inst.Status() == instance.StatusRunning {
		return inst, nil
	}

	if err := inst.Start(); err != nil {
		return nil, fmt.Errorf("start instance: %w", err)
	}

	if err := m.instanceStore.Save(inst); err != nil {
		return nil, fmt.Errorf("save instance state: %w", err)
	}

	// The process has been launched but is not yet healthy. Poll its
	// health endpoint in the background and transition the status to
	// running once it responds, or failed on timeout.
	go m.waitForInstanceHealthy(inst)

	return inst, nil
}

// waitForInstanceHealthy polls the instance's health endpoint until it
// becomes healthy or the start timeout elapses, then transitions the
// instance status accordingly and persists it.
func (m *manager) waitForInstanceHealthy(inst *instance.Instance) {
	timeout := m.cfg.Instances.StartTimeout
	if err := inst.WaitForHealthy(timeout); err != nil {
		inst.MarkFailed()
		log.Printf("instance %s failed to become healthy: %v", inst.Name, err)
	} else {
		inst.MarkRunning(inst.Port)
	}
	if err := m.instanceStore.Save(inst); err != nil {
		log.Printf("save instance %s state: %v", inst.Name, err)
	}
}

func (m *manager) StopInstance(name string) (*instance.Instance, error) {
	m.mu.Lock()
	inst, ok := m.registry.Get(name)
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("instance %q not found", name)
	}

	if inst.Status() == instance.StatusStopped {
		return inst, nil
	}

	if err := inst.Stop(); err != nil {
		return nil, fmt.Errorf("stop instance: %w", err)
	}

	m.portAllocator.Free(inst.Port)
	m.registry.markStopped(name)

	if err := m.instanceStore.Save(inst); err != nil {
		return nil, fmt.Errorf("save instance state: %w", err)
	}

	return inst, nil
}

func (m *manager) RestartInstance(name string) (*instance.Instance, error) {
	m.mu.Lock()
	inst, ok := m.registry.Get(name)
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("instance %q not found", name)
	}

	if err := inst.Restart(); err != nil {
		return nil, fmt.Errorf("restart instance: %w", err)
	}
	if err := m.instanceStore.Save(inst); err != nil {
		return nil, fmt.Errorf("save instance state: %w", err)
	}

	go m.waitForInstanceHealthy(inst)

	return inst, nil
}

func (m *manager) DeleteInstance(name string) error {
	m.mu.Lock()
	inst, ok := m.registry.Get(name)
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("instance %q not found", name)
	}

	if inst.Status() == instance.StatusRunning {
		if err := inst.Stop(); err != nil {
			return fmt.Errorf("stop instance before delete: %w", err)
		}
		m.portAllocator.Free(inst.Port)
	}

	m.registry.Delete(name)

	if err := m.instanceStore.Delete(name); err != nil {
		return fmt.Errorf("delete instance: %w", err)
	}

	return nil
}

func (m *manager) UpdateInstance(name string, opts *instance.Options) (*instance.Instance, error) {
	m.mu.Lock()
	inst, ok := m.registry.Get(name)
	m.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("instance %q not found", name)
	}

	if inst.Status() == instance.StatusRunning {
		return nil, fmt.Errorf("cannot update running instance; stop it first")
	}

	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("validate options: %w", err)
	}

	inst.Opts = opts
	inst.UpdatedAt = time.Now().Unix()

	if err := m.instanceStore.Save(inst); err != nil {
		return nil, fmt.Errorf("save instance: %w", err)
	}

	return inst, nil
}

func (m *manager) GetInstanceLogs(name string, lines int) (string, error) {
	inst, ok := m.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("instance %q not found", name)
	}

	return inst.Logs(lines)
}
