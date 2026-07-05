package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanModels_AssignsAliases(t *testing.T) {
	dir := t.TempDir()
	mkdir := func(p string) {
		if err := os.MkdirAll(filepath.Join(dir, p), 0755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p string) {
		full := filepath.Join(dir, p)
		if err := os.WriteFile(full, []byte("gguf"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Single model: alias should equal the bare filename stem.
	mkdir("unsloth--Qwen3.5-9B-GGUF")
	write("unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf")

	// Two models with the same filename in different repos -> disambiguated
	// by prepending the parent directory name.
	mkdir("repo-A")
	write("repo-A/model.gguf")
	mkdir("repo-B")
	write("repo-B/model.gguf")

	models, err := ScanModels(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(models) != 3 {
		t.Fatalf("expected 3 models, got %d", len(models))
	}

	byPath := map[string]string{}
	for _, m := range models {
		byPath[m.Path] = m.Alias
		if m.Alias == "" {
			t.Errorf("model %q has empty alias", m.Name)
		}
	}

	qwen := filepath.Join(dir, "unsloth--Qwen3.5-9B-GGUF", "Qwen3.5-9B-Q4_K_M.gguf")
	if got := byPath[qwen]; got != "Qwen3.5-9B-Q4_K_M" {
		t.Errorf("alias for qwen = %q, want Qwen3.5-9B-Q4_K_M", got)
	}

	// Disambiguated aliases must be distinct and non-empty.
	a := byPath[filepath.Join(dir, "repo-A", "model.gguf")]
	b := byPath[filepath.Join(dir, "repo-B", "model.gguf")]
	if a == "" || b == "" {
		t.Fatalf("disambiguated aliases empty: a=%q b=%q", a, b)
	}
	if a == b {
		t.Fatalf("aliases collided: %q", a)
	}
}

func TestScanModels_NoModels(t *testing.T) {
	dir := t.TempDir()
	models, err := ScanModels(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("expected 0 models, got %d", len(models))
	}
}
