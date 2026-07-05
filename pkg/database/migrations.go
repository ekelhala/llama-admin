package database

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func RunMigrations(dbPath string) error {
	dir, err := extractMigrations()
	if err != nil {
		return fmt.Errorf("extract migrations: %w", err)
	}
	defer os.RemoveAll(dir)

	m, err := migrate.New("file://"+dir, "sqlite3://"+dbPath)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("database is already up to date")
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Println("database migrations applied")
	return nil
}

func extractMigrations() (string, error) {
	dir, err := os.MkdirTemp("", "migrate-*")
	if err != nil {
		return "", err
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			os.RemoveAll(dir)
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0644); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}

	return dir, nil
}
