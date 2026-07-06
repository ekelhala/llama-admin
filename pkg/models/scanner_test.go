package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanModelFiles_FindsGgufFiles(t *testing.T) {
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

	// Single model
	mkdir("unsloth--Qwen3.5-9B-GGUF")
	write("unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf")

	// Two models in different repos
	mkdir("repo-A")
	write("repo-A/model.gguf")
	mkdir("repo-B")
	write("repo-B/model.gguf")

	files, err := ScanModelFiles(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 models, got %d", len(files))
	}

	byPath := map[string]ModelFileInfo{}
	for _, m := range files {
		byPath[m.Path] = m
	}

	qwen := filepath.Join(dir, "unsloth--Qwen3.5-9B-GGUF", "Qwen3.5-9B-Q4_K_M.gguf")
	if got := byPath[qwen].Name; got != "unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf" {
		t.Errorf("name for qwen = %q, want unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf", got)
	}
}

func TestScanModelFiles_NoModels(t *testing.T) {
	dir := t.TempDir()
	files, err := ScanModelFiles(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 models, got %d", len(files))
	}
}

func TestScanModelFiles_NestedDirectories(t *testing.T) {
	dir := t.TempDir()
	mk := func(p string) {
		if err := os.MkdirAll(filepath.Join(dir, p), 0755); err != nil {
			t.Fatal(err)
		}
	}
	wr := func(p string) {
		if err := os.WriteFile(filepath.Join(dir, p), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mk("a/b/c")
	wr("a/b/c/deep.gguf")
	mk("flat")
	wr("flat/top.gguf")

	files, err := ScanModelFiles(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 models, got %d", len(files))
	}

	// SizeBytes must reflect the file content length.
	for _, m := range files {
		if m.SizeBytes != 1 {
			t.Errorf("SizeBytes for %s = %d, want 1", m.Name, m.SizeBytes)
		}
		if m.Source != "scan" {
			t.Errorf("Source for %s = %q, want scan", m.Name, m.Source)
		}
	}
}
