package database

import (
	"database/sql"
	"fmt"
	"time"

	"llama-admin/pkg/auth"
)

type APIKeyStore struct {
	db *sql.DB
}

func NewAPIKeyStore(db *sql.DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

type APIKey struct {
	ID             int64     `json:"id"`
	Key            string    `json:"key,omitempty"`
	KeyHash        string    `json:"-"`
	Name           string    `json:"name"`
	UserID         *int64    `json:"user_id"`
	PermissionMode string    `json:"permission_mode"`
	ExpiresAt      *int64    `json:"expires_at"`
	CreatedAt      int64     `json:"created_at"`
	UpdatedAt      int64     `json:"updated_at"`
	LastUsedAt     *int64    `json:"last_used_at"`
}

func (s *APIKeyStore) Create(name string, userID *int64, permissionMode string, expiresAt *int64) (*APIKey, error) {
	if permissionMode == "" {
		permissionMode = "per_instance"
	}

	plainKey, err := auth.GenerateKey("la")
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}

	hash, err := auth.HashKey(plainKey)
	if err != nil {
		return nil, fmt.Errorf("hash key: %w", err)
	}

	now := time.Now().Unix()
	result, err := s.db.Exec(`
		INSERT INTO api_keys (key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, hash, name, userID, permissionMode, expiresAt, now, now)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &APIKey{
		ID:             id,
		Key:            plainKey,
		KeyHash:        hash,
		Name:           name,
		UserID:         userID,
		PermissionMode: permissionMode,
		ExpiresAt:      expiresAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s *APIKeyStore) List() ([]APIKey, error) {
	rows, err := s.db.Query(`
		SELECT id, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var hash string
		if err := rows.Scan(&k.ID, &hash, &k.Name, &k.UserID, &k.PermissionMode,
			&k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		k.KeyHash = hash
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *APIKeyStore) Get(id int64) (*APIKey, error) {
	var k APIKey
	err := s.db.QueryRow(`
		SELECT id, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys WHERE id = $1
	`, id).Scan(&k.ID, &k.Name, &k.UserID, &k.PermissionMode,
		&k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt, &k.LastUsedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("api key %d not found", id)
	}
	return &k, err
}

func (s *APIKeyStore) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = $1", id)
	return err
}

func (s *APIKeyStore) GetActiveKeys() ([]APIKey, error) {
	rows, err := s.db.Query(`
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE expires_at IS NULL OR expires_at > strftime('%s', 'now')
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		var hash string
		if err := rows.Scan(&k.ID, &hash, &k.Name, &k.UserID, &k.PermissionMode,
			&k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (s *APIKeyStore) TouchKey(id int64) {
	_, err := s.db.Exec(`
		UPDATE api_keys SET last_used_at = strftime('%s', 'now') WHERE id = $1
	`, id)
	if err != nil {
		// Best-effort, log but don't fail
	}
}

func (s *APIKeyStore) GetByKeyHash(keyHash string) (*APIKey, error) {
	var k APIKey
	var hash string
	err := s.db.QueryRow(`
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys WHERE key_hash = $1
	`, keyHash).Scan(&k.ID, &hash, &k.Name, &k.UserID, &k.PermissionMode,
		&k.ExpiresAt, &k.CreatedAt, &k.UpdatedAt, &k.LastUsedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("api key not found")
	}
	return &k, err
}
