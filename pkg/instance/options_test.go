package instance

import (
	"testing"
)

func TestOptionsUnmarshalJSON_BackendOptionsFromMap(t *testing.T) {
	// Regression test: when JSON is decoded via Options.UnmarshalJSON, the
	// nested backend_options object arrives as map[string]any (not
	// json.RawMessage). The handler must still populate BackendOptions so
	// that backend validation can find the "model" field.
	body := []byte(`{"backend_type":"llama_cpp","backend_options":{"model":"/tmp/model.gguf","ctx_size":4096}}`)

	var o Options
	if err := o.UnmarshalJSON(body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if err := o.ValidateAndApplyDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if got, _ := o.BackendOptions["model"].(string); got != "/tmp/model.gguf" {
		t.Fatalf("model = %q, want /tmp/model.gguf; BackendOptions=%v", got, o.BackendOptions)
	}
}

func TestOptionsUnmarshalJSON_RoundTrip(t *testing.T) {
	body := []byte(`{"backend_type":"llama_cpp","backend_options":{"model":"/tmp/x.gguf","n_gpu_layers":4}}`)

	var o Options
	if err := o.UnmarshalJSON(body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := o.ValidateAndApplyDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	out, err := o.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(out), `"/tmp/x.gguf"`) {
		t.Fatalf("marshalled output missing model: %s", out)
	}
	if !contains(string(out), `"n_gpu_layers":4`) {
		t.Fatalf("marshalled output missing n_gpu_layers: %s", out)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
