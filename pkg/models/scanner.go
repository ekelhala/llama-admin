package models

import (
	"os"
	"path/filepath"
)

type ModelFileInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"size_bytes"`
	Source     string `json:"source"` // "scan" or "download"
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

	return models, nil
}
