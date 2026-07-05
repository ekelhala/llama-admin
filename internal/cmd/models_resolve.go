package cmd

import (
	"encoding/json"
	"fmt"

	client "llama-admin/internal/client"
	"llama-admin/pkg/models"
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
	if arg == "" {
		return "", nil
	}

	data, err := c.Get("/api/v1/models")
	if err != nil {
		// Don't fail instance creation just because the catalog is
		// unreachable; fall back to passing the arg through unchanged.
		return "", nil
	}

	var rawModels []map[string]any
	if err := json.Unmarshal(data, &rawModels); err != nil {
		return "", fmt.Errorf("parse model catalog: %w", err)
	}

	catalog := make([]models.ModelFileInfo, 0, len(rawModels))
	for _, m := range rawModels {
		path, _ := m["path"].(string)
		if path == "" {
			continue
		}
		catalog = append(catalog, models.ModelFileInfo{
			Name:  asString(m["name"]),
			Alias: asString(m["alias"]),
			Path:  path,
		})
	}

	return models.ResolveModelArg(catalog, arg), nil
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
