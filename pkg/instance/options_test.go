package instance

import (
	"encoding/json"
	"testing"
)

func TestOptionsValidate_RequiredModelAlias(t *testing.T) {
	o := &Options{Params: map[string]string{}}
	err := o.Validate()
	if err == nil {
		t.Fatal("expected error for missing model_alias")
	}
}

func TestOptionsValidate_Valid(t *testing.T) {
	o := &Options{
		ModelAlias: "qwen3-9b",
		Params:     map[string]string{"ctx-size": "8192"},
		Env:        map[string]string{"CUDA_VISIBLE_DEVICES": "0"},
	}
	err := o.Validate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOptionsValidate_BlockedParam(t *testing.T) {
	o := &Options{
		ModelAlias: "qwen3-9b",
		Params:     map[string]string{"model": "/path/to/model.gguf"},
	}
	err := o.Validate()
	if err == nil {
		t.Fatal("expected error for blocked param 'model'")
	}
}

func TestOptions_RoundTrip(t *testing.T) {
	autoRestart := true
	original := &Options{
		ModelAlias:  "qwen3-9b",
		Params:      map[string]string{"ctx-size": "8192", "flash-attn": ""},
		Env:         map[string]string{"KEY": "val"},
		AutoRestart: &autoRestart,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Options
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ModelAlias != original.ModelAlias {
		t.Errorf("model_alias = %q, want %q", decoded.ModelAlias, original.ModelAlias)
	}
	if decoded.Params["ctx-size"] != "8192" {
		t.Errorf("params[ctx-size] = %q, want 8192", decoded.Params["ctx-size"])
	}
	if decoded.AutoRestart == nil || *decoded.AutoRestart != true {
		t.Errorf("auto_restart = %v, want true", decoded.AutoRestart)
	}
}
