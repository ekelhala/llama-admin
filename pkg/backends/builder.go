package backends

import (
	"fmt"
)

var backendConstructors = map[BackendType]func() Backend{
	BackendTypeLlamaCpp: func() Backend {
		return &LlamaServerBackend{}
	},
}

type Backend interface {
	BuildCommandArgs(binaryPath string, opts BackendOptions) ([]string, error)
	Validate(opts BackendOptions) error
}

func GetConstructor(t BackendType) (func() Backend, bool) {
	f, ok := backendConstructors[t]
	return f, ok
}

func NewBackend(t BackendType) (Backend, error) {
	constructor, ok := backendConstructors[t]
	if !ok {
		return nil, fmt.Errorf("unknown backend type: %s", t)
	}
	return constructor(), nil
}

func (o *Options) Validate() error {
	if o.BackendType == "" {
		return fmt.Errorf("backend_type is required")
	}
	ctor, ok := backendConstructors[o.BackendType]
	if !ok {
		return fmt.Errorf("unsupported backend type: %s", o.BackendType)
	}
	b := ctor()
	if o.BackendOptions == nil {
		o.BackendOptions = make(BackendOptions)
	}
	return b.Validate(o.BackendOptions)
}
