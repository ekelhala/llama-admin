package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	client "llama-admin/internal/client"
)

// resolveModelArg resolves a user-supplied --model value to a concrete
// filesystem path using the server's model catalog.
//
// Matching is performed against, in order:
//  1. The model alias (e.g. "Qwen3.5-9B-Q4_K_M")
//  2. The model name (the relative path returned by the scanner)
//  3. The bare filename (with or without the .gguf extension)
//  4. The absolute path
//
// An empty arg, or an arg that does not match any catalog entry, returns an
// empty string so the caller can fall back to passing the value through
// unchanged (preserving backward compatibility with absolute paths).
func resolveModelArg(c *client.Client, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", nil
	}

	data, err := c.Get("/api/v1/models")
	if err != nil {
		// Don't fail instance creation just because the catalog is
		// unreachable; fall back to passing the arg through unchanged.
		return "", nil
	}

	var models []map[string]any
	if err := json.Unmarshal(data, &models); err != nil {
		return "", fmt.Errorf("parse model catalog: %w", err)
	}

	argNorm := normalizeModelRef(arg)
	for _, m := range models {
		path, _ := m["path"].(string)
		if path == "" {
			continue
		}
		if path == arg {
			return path, nil
		}
		for _, ref := range []string{
			asString(m["alias"]),
			asString(m["name"]),
			filepath.Base(asString(m["name"])),
			strings.TrimSuffix(filepath.Base(asString(m["name"])), ".gguf"),
			filepath.Base(path),
			strings.TrimSuffix(filepath.Base(path), ".gguf"),
		} {
			if ref == "" {
				continue
			}
			if normalizeModelRef(ref) == argNorm {
				return path, nil
			}
		}
	}

	return "", nil
}

// normalizeModelRef lowercases and trims surrounding whitespace so that
// alias matching is forgiving of case differences in the CLI argument.
// Hyphens and underscores are collapsed to the same character because
// model filenames conventionally use underscores (e.g. "Q4_K_M") while
// users frequently type the hyphenated form (e.g. "Q4-K-M"); treating them
// as equivalent lets resolution succeed regardless of which style is used.
func normalizeModelRef(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	b := []byte(s)
	for i, c := range b {
		if c == '-' {
			b[i] = '_'
		}
	}
	return string(b)
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
