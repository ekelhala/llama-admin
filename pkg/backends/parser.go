package backends

import (
	"encoding/json"
	"fmt"
	"strings"
)

func marshalFlat(m map[string]any) ([]byte, error) {
	return json.Marshal(m)
}

func unmarshalFlat(data []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func SplitInstanceModel(model string) (instanceName, modelName string) {
	parts := strings.SplitN(model, "/", 2)
	instanceName = parts[0]
	if len(parts) > 1 {
		modelName = parts[1]
	}
	return
}

func ValidateInstanceName(name string) error {
	if name == "" {
		return fmt.Errorf("instance name is required")
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return fmt.Errorf("invalid instance name: %q", name)
	}
	return nil
}
