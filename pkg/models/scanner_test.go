package models

import (
	"os"
	"path/filepath"
	"strings"
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

func TestScanModels_NestedDirectories(t *testing.T) {
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

	models, err := ScanModels(dir)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	// SizeBytes must reflect the file content length.
	for _, m := range models {
		if m.SizeBytes != 1 {
			t.Errorf("SizeBytes for %s = %d, want 1", m.Name, m.SizeBytes)
		}
		if m.Source != "scan" {
			t.Errorf("Source for %s = %q, want scan", m.Name, m.Source)
		}
	}
}

func TestAssignAliases_NoCollisions(t *testing.T) {
	models := []ModelFileInfo{
		{Name: "alpha.gguf", Path: "/cache/alpha.gguf"},
		{Name: "beta.gguf", Path: "/cache/beta.gguf"},
		{Name: "gamma.gguf", Path: "/cache/gamma.gguf"},
	}
	assignAliases(models)

	seen := map[string]bool{}
	for _, m := range models {
		if m.Alias == "" {
			t.Errorf("alias for %s is empty", m.Name)
		}
		if seen[m.Alias] {
			t.Errorf("alias %q collided", m.Alias)
		}
		seen[m.Alias] = true
	}
	if models[0].Alias != "alpha" {
		t.Errorf("alpha alias = %q, want alpha", models[0].Alias)
	}
}

func TestAssignAliases_DisambiguatesByParentDir(t *testing.T) {
	models := []ModelFileInfo{
		{Name: "repoA/model.gguf", Path: "/cache/repoA/model.gguf"},
		{Name: "repoB/model.gguf", Path: "/cache/repoB/model.gguf"},
	}
	assignAliases(models)

	if models[0].Alias == models[1].Alias {
		t.Fatalf("aliases collided: %q", models[0].Alias)
	}
	if !strings.Contains(models[0].Alias, "repoA") {
		t.Errorf("alias %q should contain repoA", models[0].Alias)
	}
	if !strings.Contains(models[1].Alias, "repoB") {
		t.Errorf("alias %q should contain repoB", models[1].Alias)
	}
}

func TestAssignAliases_NumericSuffixWhenParentDirAlsoCollides(t *testing.T) {
	// Two models in different but same-named parent directories would
	// produce identical "parent/stem" aliases; the registry must add a
	// numeric suffix.
	models := []ModelFileInfo{
		{Name: "repo/model.gguf", Path: "/cache/repo/model.gguf"},
		{Name: "repo/model.gguf", Path: "/other/repo/model.gguf"},
	}
	assignAliases(models)

	if models[0].Alias == models[1].Alias {
		t.Fatalf("aliases still collided: %q", models[0].Alias)
	}
}

func TestSanitizeAliasToken(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"simple", "simple"},
		{"has space", "has-space"},
		{"tab\there", "tab-here"},
		{"with/slash", "with-slash"},
		{"back\\slash", "back-slash"},
		{"col:on", "col-on"},
		{"  trim  ", "trim"},
		{"newline\nhere", "newline-here"},
	}
	for _, c := range cases {
		if got := sanitizeAliasToken(c.in); got != c.want {
			t.Errorf("sanitizeAliasToken(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
