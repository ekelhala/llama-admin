package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"llama-admin/pkg/config"
)

func Open(cfg *config.AppConfig) (*sql.DB, error) {
	dsn := cfg.Database.Path + "?_busy_timeout=5000&_journal_mode=WAL"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("database opened: %s", cfg.Database.Path)
	return db, nil
}
