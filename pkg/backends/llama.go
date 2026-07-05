package backends

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type LlamaServerBackend struct{}

type LlamaServerOptions struct {
	Model      string `json:"model"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	CtxSize    int    `json:"ctx_size"`
	NGpuLayers int    `json:"n_gpu_layers"`
}

func (b *LlamaServerBackend) BuildCommandArgs(binaryPath string, opts BackendOptions) ([]string, error) {
	var lo LlamaServerOptions
	data, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("marshal options: %w", err)
	}
	if err := json.Unmarshal(data, &lo); err != nil {
		return nil, fmt.Errorf("unmarshal options: %w", err)
	}

	args := []string{fmt.Sprintf("--model=%s", lo.Model)}

	host := lo.Host
	if host == "" {
		host = "127.0.0.1"
	}
	args = append(args, fmt.Sprintf("--host=%s", host))

	if lo.Port != 0 {
		args = append(args, fmt.Sprintf("--port=%d", lo.Port))
	}

	if lo.CtxSize != 0 {
		args = append(args, fmt.Sprintf("--ctx-size=%d", lo.CtxSize))
	}

	if lo.NGpuLayers != 0 {
		args = append(args, fmt.Sprintf("--n-gpu-layers=%d", lo.NGpuLayers))
	}

	return args, nil
}

func (b *LlamaServerBackend) Validate(opts BackendOptions) error {
	model, ok := opts["model"]
	if !ok || model == "" {
		return fmt.Errorf("model is required for llama_cpp backend")
	}
	return nil
}

func (o *LlamaServerOptions) GetModel() string  { return o.Model }
func (o *LlamaServerOptions) GetHost() string   { return o.Host }
func (o *LlamaServerOptions) GetPort() int      { return o.Port }

func (o *LlamaServerOptions) Validate() error {
	if o.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

func (o *LlamaServerOptions) BuildCommandArgs(binaryPath string) []string {
	var args []string
	args = append(args, binaryPath)
	args = append(args, fmt.Sprintf("--model=%s", o.Model))

	host := o.Host
	if host == "" {
		host = "127.0.0.1"
	}
	args = append(args, fmt.Sprintf("--host=%s", host))

	if o.Port != 0 {
		args = append(args, fmt.Sprintf("--port=%d", o.Port))
	}

	if o.CtxSize != 0 {
		args = append(args, fmt.Sprintf("--ctx-size=%d", o.CtxSize))
	}

	if o.NGpuLayers != 0 {
		args = append(args, fmt.Sprintf("--n-gpu-layers=%d", o.NGpuLayers))
	}

	return args
}

func (o *LlamaServerOptions) ParseCommand(args []string) error {
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--model":
			o.Model = args[i+1]
		case "--host":
			o.Host = args[i+1]
		case "--port":
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid port: %w", err)
			}
			o.Port = v
		case "--ctx-size":
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid ctx-size: %w", err)
			}
			o.CtxSize = v
		case "--n-gpu-layers":
			v, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid n-gpu-layers: %w", err)
			}
			o.NGpuLayers = v
		}
	}
	return nil
}
