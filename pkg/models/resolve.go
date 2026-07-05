package models

import (
	"path/filepath"
	"strings"
)

// ResolveModelArg resolves a user-supplied model reference to a concrete
// on-disk path using the scanned model catalog.
//
// Matching is performed against, in order:
//  1. The model alias (e.g. "Qwen3.5-9B-Q4_K_M")
//  2. The model name (the relative path returned by the scanner)
//  3. The bare filename (with or without the .gguf extension)
//  4. The absolute path
//
// An empty arg, or an arg that does not match any catalog entry, returns
// an empty string so the caller can fall back to passing the value
// through unchanged (preserving backward compatibility with absolute
// paths that live outside the catalog).
func ResolveModelArg(catalog []ModelFileInfo, arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return ""
	}

	argNorm := NormalizeModelRef(arg)
	for _, m := range catalog {
		if m.Path == arg {
			return m.Path
		}
		for _, ref := range []string{
			m.Alias,
			m.Name,
			filepath.Base(m.Name),
			strings.TrimSuffix(filepath.Base(m.Name), ".gguf"),
			filepath.Base(m.Path),
			strings.TrimSuffix(filepath.Base(m.Path), ".gguf"),
		} {
			if ref == "" {
				continue
			}
			if NormalizeModelRef(ref) == argNorm {
				return m.Path
			}
		}
	}

	return ""
}

// NormalizeModelRef lowercases and trims surrounding whitespace so that
// alias matching is forgiving of case differences in the CLI argument.
// Hyphens and underscores are collapsed to the same character because
// model filenames conventionally use underscores (e.g. "Q4_K_M") while
// users frequently type the hyphenated form (e.g. "Q4-K-M"); treating
// them as equivalent lets resolution succeed regardless of which style
// is used.
func NormalizeModelRef(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	b := []byte(s)
	for i, c := range b {
		if c == '-' {
			b[i] = '_'
		}
	}
	return string(b)
}
