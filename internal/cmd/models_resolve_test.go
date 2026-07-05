package cmd

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
			if got := normalizeModelRef(c.in); got != c.want {
				t.Errorf("normalizeModelRef(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}

	// The motivating bug: the hyphenated and underscore forms of a quant
	// name must resolve to the same normalized key.
	if normalizeModelRef("Qwen3.5-9B-Q4-K-M") != normalizeModelRef("Qwen3.5-9B-Q4_K_M") {
		t.Fatal("hyphenated and underscore quant names must normalize equally")
	}
}
