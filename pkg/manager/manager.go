package manager

import (
	"database/sql"
	"log"
	"sync"

	"llama-admin/pkg/config"
	"llama-admin/pkg/database"
	"llama-admin/pkg/instance"
	"llama-admin/pkg/models"
)

type InstanceManager interface {
	CreateInstance(name string, opts *instance.Options) (*instance.Instance, error)
	GetInstance(name string) (*instance.Instance, error)
	ListInstances() ([]*instance.Instance, error)
	StartInstance(name string) (*instance.Instance, error)
	StopInstance(name string) (*instance.Instance, error)
	RestartInstance(name string) (*instance.Instance, error)
	DeleteInstance(name string) error
	UpdateInstance(name string, opts *instance.Options) (*instance.Instance, error)
	GetInstanceLogs(name string, lines int) (string, error)
	Shutdown()
}

type manager struct {
	mu            sync.RWMutex
	registry      *registry
	portAllocator *portAllocator
	db            *sql.DB
	instanceStore *database.InstanceStore
	cfg           *config.AppConfig
	modelMgr      *models.Manager
}

func New(cfg *config.AppConfig, db *sql.DB, modelMgr *models.Manager) *manager {
	m := &manager{
		registry:      newRegistry(),
		portAllocator: newPortAllocator(cfg.Instances.PortRange.Min, cfg.Instances.PortRange.Max),
		db:            db,
		instanceStore: database.NewInstanceStore(db),
		cfg:           cfg,
		modelMgr:      modelMgr,
	}

	m.loadInstances()
	return m
}

func (m *manager) loadInstances() {
	instances, err := m.instanceStore.LoadAll()
	if err != nil {
		log.Printf("warning: failed to load instances: %v", err)
		return
	}

	for _, inst := range instances {
		// Recreate instance with internal fields
		newInst, err := instance.New(inst.ID, inst.Name, inst.Opts, "127.0.0.1", 0, m.cfg)
		if err != nil {
			log.Printf("warning: failed to recreate instance %s: %v", inst.Name, err)
			continue
		}
		newInst.ID = inst.ID
		newInst.CreatedAt = inst.CreatedAt
		newInst.UpdatedAt = inst.UpdatedAt
		newInst.OwnerUserID = inst.OwnerUserID

		// Restore port
		if inst.Opts != nil && inst.Opts.BackendOptions != nil {
			if port, ok := inst.Opts.BackendOptions["port"]; ok {
				if p, ok := port.(int); ok && p != 0 {
					m.portAllocator.MarkAllocated(p, inst.Name)
					newInst.Port = p
				}
			}
		}

		m.registry.Add(newInst)

		// Auto-restart if needed
		if inst.Status() == instance.StatusRunning {
			if inst.Opts != nil && inst.Opts.AutoRestart != nil && *inst.Opts.AutoRestart {
				go func(name string) {
					if _, err := m.StartInstance(name); err != nil {
						log.Printf("auto-restart failed for %s: %v", name, err)
					}
				}(inst.Name)
			}
		}
	}

	log.Printf("loaded %d instances", len(instances))
}

func (m *manager) Shutdown() {
	instances := m.registry.List()
	var wg sync.WaitGroup
	for _, inst := range instances {
		if inst.Status() == instance.StatusRunning {
			wg.Add(1)
			go func(inst *instance.Instance) {
				defer wg.Done()
				if err := inst.Stop(); err != nil {
					log.Printf("error stopping instance %s: %v", inst.Name, err)
				}
			}(inst)
		}
	}
	wg.Wait()
	log.Println("all instances stopped")
}
