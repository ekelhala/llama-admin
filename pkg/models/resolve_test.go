package models

import "testing"

func TestNormalizeModelRef(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "whitespace trimmed", in: "  Qwen3.5-9B  ", want: "qwen3.5_9b"},
		{name: "lowercased", in: "Qwen3.5-9B", want: "qwen3.5_9b"},
		{name: "underscores preserved", in: "Qwen3.5-9B-Q4_K_M", want: "qwen3.5_9b_q4_k_m"},
		{name: "hyphens collapsed to underscores", in: "Qwen3.5-9B-Q4-K-M", want: "qwen3.5_9b_q4_k_m"},
		{name: "mixed separators equal", in: "Qwen3.5-9B-Q4_K-M", want: "qwen3.5_9b_q4_k_m"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeModelRef(c.in); got != c.want {
				t.Errorf("NormalizeModelRef(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	// The motivating bug: the hyphenated and underscore forms of a quant
	// name must normalize to the same key.
	if NormalizeModelRef("Qwen3.5-9B-Q4-K-M") != NormalizeModelRef("Qwen3.5-9B-Q4_K_M") {
		t.Fatal("hyphenated and underscore quant names must normalize equally")
	}
}

func TestResolveModelArg(t *testing.T) {
	catalog := []ModelFileInfo{
		{
			Name:  "unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf",
			Alias: "Qwen3.5-9B-Q4_K_M",
			Path:  "/var/lib/llama-admin/models/unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf",
		},
	}

	cases := []struct {
		name string
		arg  string
		want string
	}{
		{name: "empty passthrough", arg: "", want: ""},
		{name: "exact path", arg: catalog[0].Path, want: catalog[0].Path},
		{name: "alias with underscores", arg: "Qwen3.5-9B-Q4_K_M", want: catalog[0].Path},
		{name: "alias with hyphens", arg: "Qwen3.5-9B-Q4-K-M", want: catalog[0].Path},
		{name: "alias case-insensitive", arg: "qwen3.5-9b-q4_k_m", want: catalog[0].Path},
		{name: "bare filename with extension", arg: "Qwen3.5-9B-Q4_K_M.gguf", want: catalog[0].Path},
		{name: "bare filename hyphenated no extension", arg: "Qwen3.5-9B-Q4-K-M", want: catalog[0].Path},
		{name: "relative name", arg: "unsloth--Qwen3.5-9B-GGUF/Qwen3.5-9B-Q4_K_M.gguf", want: catalog[0].Path},
		{name: "unknown returns empty", arg: "does-not-exist", want: ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveModelArg(catalog, c.arg); got != c.want {
				t.Errorf("ResolveModelArg(%q) = %q, want %q", c.arg, got, c.want)
			}
		})
	}
}
