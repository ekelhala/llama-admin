package manager

import (
	"sync"

	"llama-admin/pkg/instance"
)

type registry struct {
	mu        sync.RWMutex
	instances sync.Map // string -> *instance.Instance
}

func newRegistry() *registry {
	return &registry{}
}

func (r *registry) Add(inst *instance.Instance) {
	r.instances.Store(inst.Name, inst)
}

func (r *registry) Get(name string) (*instance.Instance, bool) {
	v, ok := r.instances.Load(name)
	if !ok {
		return nil, false
	}
	return v.(*instance.Instance), true
}

func (r *registry) Delete(name string) {
	r.instances.Delete(name)
}

func (r *registry) List() []*instance.Instance {
	var result []*instance.Instance
	r.instances.Range(func(_, value any) bool {
		result = append(result, value.(*instance.Instance))
		return true
	})
	return result
}

func (r *registry) markStopped(name string) {
	if inst, ok := r.Get(name); ok {
		inst.MarkStopped()
	}
}
