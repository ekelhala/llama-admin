package models

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ModelFileInfo struct {
	Name      string `json:"name"`
	Alias     string `json:"alias"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Source    string `json:"source"` // "scan" or "download"
}

func ScanModels(cacheDir string) ([]ModelFileInfo, error) {
	var models []ModelFileInfo

	if err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".gguf" {
			relPath, _ := filepath.Rel(cacheDir, path)
			models = append(models, ModelFileInfo{
				Name:      relPath,
				Path:      path,
				SizeBytes: info.Size(),
				Source:    "scan",
			})
		}
		return nil
	}); err != nil {
		return nil, err
	}

	assignAliases(models)
	return models, nil
}

// assignAliases populates the Alias field of each model with a short, unique
// identifier derived from the model filename (the .gguf extension stripped).
// When two models would yield the same alias, the parent directory name is
// prepended to disambiguate; if that still collides, a numeric suffix is added.
func assignAliases(models []ModelFileInfo) {
	used := make(map[string]int, len(models))

	stems := make([]string, len(models))
	for i := range models {
		stems[i] = sanitizeAliasToken(strings.TrimSuffix(filepath.Base(models[i].Name), ".gguf"))
	}

	collisions := make(map[string]int)
	for _, s := range stems {
		collisions[s]++
	}

	for i, stem := range stems {
		final := stem
		if collisions[stem] > 1 {
			parent := sanitizeAliasToken(filepath.Base(filepath.Dir(models[i].Path)))
			if parent != "" && parent != "." && parent != "-" {
				final = parent + "/" + stem
			}
		}
		if n, ok := used[final]; ok {
			used[final] = n + 1
			final = final + "-" + strconv.Itoa(n+1)
		} else {
			used[final] = 1
		}
		models[i].Alias = final
	}
}

// sanitizeAliasToken keeps an alias token readable by replacing whitespace and
// path separators with hyphens.
func sanitizeAliasToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	r := []rune(s)
	for i, c := range r {
		switch c {
		case ' ', '\t', '\n', '\r', '/', '\\', ':':
			r[i] = '-'
		}
	}
	return string(r)
}
