package config

import (
	"os"
	"testing"
)

func TestExpandPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		env      map[string]string
		expected string
	}{
		{
			name:     "simple replacement",
			input:    "${TEST_HOME}",
			env:      map[string]string{"TEST_HOME": "/home/user"},
			expected: "/home/user",
		},
		{
			name:     "default when unset",
			input:    "${TEST_UNSET:-default_value}",
			env:      map[string]string{},
			expected: "default_value",
		},
		{
			name:     "no default when set",
			input:    "${TEST_HOME:-/fallback}",
			env:      map[string]string{"TEST_HOME": "/home/user"},
			expected: "/home/user",
		},
		{
			name:     "default when empty",
			input:    "${TEST_EMPTY:-fallback}",
			env:      map[string]string{"TEST_EMPTY": ""},
			expected: "fallback",
		},
		{
			name:     "multiple placeholders",
			input:    "${TEST_HOME}/bin:${TEST_PATH:-/usr/bin}",
			env:      map[string]string{"TEST_HOME": "/home/user"},
			expected: "/home/user/bin:/usr/bin",
		},
		{
			name:     "no placeholders",
			input:    "plain_string",
			env:      map[string]string{},
			expected: "plain_string",
		},
		{
			name:     "empty default",
			input:    "${TEST_UNSET:-}",
			env:      map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := make(map[string]string)
			for k := range tt.env {
				if v, ok := os.LookupEnv(k); ok {
					original[k] = v
				}
			}
			defer func() {
				for k, v := range original {
					os.Setenv(k, v)
				}
				for k := range tt.env {
					if _, ok := original[k]; !ok {
						os.Unsetenv(k)
					}
				}
			}()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			result := ExpandPlaceholders(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
