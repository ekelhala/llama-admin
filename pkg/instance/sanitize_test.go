package instance

import "testing"

func TestSanitizeParams_Valid(t *testing.T) {
	params := map[string]string{
		"ctx-size":        "8192",
		"n-gpu-layers":    "99",
		"flash-attn":      "",
		"tensor-parallel": "1",
	}
	args, err := SanitizeParams(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{
		"--ctx-size=8192",
		"--flash-attn",
		"--n-gpu-layers=99",
		"--tensor-parallel=1",
	}
	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, exp := range expected {
		if args[i] != exp {
			t.Errorf("args[%d] = %q, want %q", i, args[i], exp)
		}
	}
}

func TestSanitizeParams_BlockedParams(t *testing.T) {
	cases := []string{"model", "host", "port"}
	for _, blocked := range cases {
		_, err := SanitizeParams(map[string]string{blocked: "value"})
		if err == nil {
			t.Errorf("expected error for blocked param %q, got nil", blocked)
		}
	}
}

func TestSanitizeParams_InvalidKey(t *testing.T) {
	cases := []string{"UPPER", "has space", "has/dot", "", "123start"}
	for _, invalid := range cases {
		_, err := SanitizeParams(map[string]string{invalid: "value"})
		if err == nil {
			t.Errorf("expected error for invalid key %q, got nil", invalid)
		}
	}
}

func TestSanitizeParams_Empty(t *testing.T) {
	args, err := SanitizeParams(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 0 {
		t.Fatalf("expected 0 args, got %d: %v", len(args), args)
	}
}
