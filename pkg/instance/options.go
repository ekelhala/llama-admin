package instance

import (
	"fmt"
)

type Options struct {
	ModelAlias  string            `json:"model_alias"`
	Params      map[string]string `json:"params"`
	Env         map[string]string `json:"env,omitempty"`
	AutoRestart *bool             `json:"auto_restart,omitempty"`
}

func (o *Options) Validate() error {
	if o.ModelAlias == "" {
		return fmt.Errorf("model_alias is required")
	}
	if o.Params == nil {
		o.Params = make(map[string]string)
	}
	if o.Env == nil {
		o.Env = make(map[string]string)
	}
	if _, err := SanitizeParams(o.Params); err != nil {
		return fmt.Errorf("sanitize params: %w", err)
	}
	return nil
}
