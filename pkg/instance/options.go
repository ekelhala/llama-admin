package instance

import (
	"encoding/json"
	"fmt"

	"llama-admin/pkg/backends"
)

type Options struct {
	BackendType     backends.BackendType
	BackendOptions  backends.BackendOptions
	DockerEnabled   *bool
	CommandOverride string
	Environment     map[string]string
	Nodes           map[string]struct{}
	AutoRestart     *bool
	PresetIni       *string
}

func (o *Options) ValidateAndApplyDefaults() error {
	if o.BackendType == "" {
		o.BackendType = backends.BackendTypeLlamaCpp
	}
	if o.BackendOptions == nil {
		o.BackendOptions = make(backends.BackendOptions)
	}
	if o.Nodes == nil {
		o.Nodes = make(map[string]struct{})
	}
	if o.Environment == nil {
		o.Environment = make(map[string]string)
	}
	ctor, ok := backends.GetConstructor(o.BackendType)
	if !ok {
		return fmt.Errorf("unsupported backend type: %s", o.BackendType)
	}
	b := ctor()
	if err := b.Validate(o.BackendOptions); err != nil {
		return err
	}
	return nil
}

func (o *Options) MarshalJSON() ([]byte, error) {
	raw := map[string]any{
		"backend_type": o.BackendType,
	}
	if o.BackendOptions != nil && len(o.BackendOptions) > 0 {
		data, err := o.BackendOptions.MarshalJSON()
		if err != nil {
			return nil, err
		}
		raw["backend_options"] = json.RawMessage(data)
	}
	if o.DockerEnabled != nil {
		raw["docker_enabled"] = *o.DockerEnabled
	}
	if o.CommandOverride != "" {
		raw["command_override"] = o.CommandOverride
	}
	if len(o.Environment) > 0 {
		raw["environment"] = o.Environment
	}
	if len(o.Nodes) > 0 {
		raw["nodes"] = o.Nodes
	}
	if o.AutoRestart != nil {
		raw["auto_restart"] = *o.AutoRestart
	}
	if o.PresetIni != nil {
		raw["preset_ini"] = *o.PresetIni
	}
	return json.Marshal(raw)
}

func (o *Options) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if v, ok := raw["backend_type"]; ok {
		if s, ok := v.(string); ok {
			o.BackendType = backends.BackendType(s)
		}
	}

	if v, ok := raw["backend_options"]; ok {
		if b, ok := v.(json.RawMessage); ok {
			opts, err := UnmarshalFlat(b)
			if err != nil {
				return err
			}
			o.BackendOptions = opts
		}
	}

	if v, ok := raw["docker_enabled"]; ok {
		if b, ok := v.(bool); ok {
			o.DockerEnabled = &b
		}
	}

	if v, ok := raw["command_override"]; ok {
		if s, ok := v.(string); ok {
			o.CommandOverride = s
		}
	}

	if v, ok := raw["environment"]; ok {
		if m, ok := v.(map[string]any); ok {
			o.Environment = make(map[string]string)
			for k, val := range m {
				o.Environment[k] = fmt.Sprintf("%v", val)
			}
		}
	}

	if v, ok := raw["nodes"]; ok {
		if m, ok := v.(map[string]any); ok {
			o.Nodes = make(map[string]struct{})
			for k := range m {
				o.Nodes[k] = struct{}{}
			}
		}
	}

	if v, ok := raw["auto_restart"]; ok {
		if b, ok := v.(bool); ok {
			o.AutoRestart = &b
		}
	}

	if v, ok := raw["preset_ini"]; ok {
		if s, ok := v.(string); ok {
			o.PresetIni = &s
		}
	}

	return nil
}

func UnmarshalFlat(data []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
