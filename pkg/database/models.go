package database

import (
	"database/sql"
	"fmt"
	"time"
)

type ModelStore struct {
	db *sql.DB
}

func NewModelStore(db *sql.DB) *ModelStore {
	return &ModelStore{db: db}
}

func (s *ModelStore) Save(m *Model) error {
	now := time.Now().Unix()
	if m.ID == 0 {
		m.CreatedAt = now
		m.UpdatedAt = now
		result, err := s.db.Exec(
			`INSERT INTO models (alias, filename, size_bytes, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			m.Alias, m.Filename, m.SizeBytes, m.CreatedAt, m.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("create model: %w", err)
		}
		id, _ := result.LastInsertId()
		m.ID = id
		return nil
	}
	m.UpdatedAt = now
	_, err := s.db.Exec(
		`UPDATE models SET alias=$1, filename=$2, size_bytes=$3, updated_at=$4 WHERE id=$5`,
		m.Alias, m.Filename, m.SizeBytes, m.UpdatedAt, m.ID,
	)
	return err
}

func (s *ModelStore) GetByID(id int64) (*Model, error) {
	m := &Model{}
	err := s.db.QueryRow(
		`SELECT id, alias, filename, size_bytes, created_at, updated_at FROM models WHERE id=$1`,
		id,
	).Scan(&m.ID, &m.Alias, &m.Filename, &m.SizeBytes, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model %d not found", id)
	}
	return m, err
}

func (s *ModelStore) GetByAlias(alias string) (*Model, error) {
	m := &Model{}
	err := s.db.QueryRow(
		`SELECT id, alias, filename, size_bytes, created_at, updated_at FROM models WHERE alias=$1`,
		alias,
	).Scan(&m.ID, &m.Alias, &m.Filename, &m.SizeBytes, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model alias %q not found", alias)
	}
	return m, err
}

func (s *ModelStore) List() ([]*Model, error) {
	rows, err := s.db.Query(`SELECT id, alias, filename, size_bytes, created_at, updated_at FROM models ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ms []*Model
	for rows.Next() {
		m := &Model{}
		if err := rows.Scan(&m.ID, &m.Alias, &m.Filename, &m.SizeBytes, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, rows.Err()
}

func (s *ModelStore) Delete(alias string) error {
	_, err := s.db.Exec(`DELETE FROM models WHERE alias=$1`, alias)
	return err
}
