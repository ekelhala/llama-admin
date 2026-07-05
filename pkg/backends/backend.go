package backends

import (
	"encoding/json"
	"fmt"
)

type BackendType string

const (
	BackendTypeLlamaCpp BackendType = "llama_cpp"
)

type BackendOptions map[string]any

func (o BackendOptions) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}
	return marshalFlat(o)
}

func (o *BackendOptions) UnmarshalJSON(data []byte) error {
	m, err := unmarshalFlat(data)
	if err != nil {
		return err
	}
	*o = m
	return nil
}

type Options struct {
	BackendType     BackendType   `json:"backend_type"`
	BackendOptions  BackendOptions `json:"backend_options,omitempty"`
	DockerEnabled   *bool         `json:"docker_enabled,omitempty"`
	CommandOverride string        `json:"command_override,omitempty"`
	Environment     map[string]string `json:"environment,omitempty"`
	Nodes           map[string]struct{} `json:"nodes,omitempty"`
	AutoRestart     *bool         `json:"auto_restart,omitempty"`
	PresetIni       *string       `json:"preset_ini,omitempty"`
}

func (o *Options) MarshalJSON() ([]byte, error) {
	type Alias Options
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
			o.BackendType = BackendType(s)
		}
	}

	if v, ok := raw["backend_options"]; ok {
		if b, ok := v.(json.RawMessage); ok {
			opts, err := unmarshalFlat(b)
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
