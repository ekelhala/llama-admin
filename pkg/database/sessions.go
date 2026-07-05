package database

import (
	"database/sql"
	"fmt"
	"time"
)

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

type Session struct {
	ID        int64     `json:"id"`
	TokenHash string    `json:"token_hash"`
	UserID    int64     `json:"user_id"`
	Provider  string    `json:"provider"`
	ExpiresAt int64     `json:"expires_at"`
	CreatedAt int64     `json:"created_at"`
	LastUsedAt *int64   `json:"last_used_at"`
}

func (s *SessionStore) Create(userID int64, provider, tokenHash string, ttl time.Duration) (*Session, error) {
	now := time.Now().Unix()
	expiresAt := now + int64(ttl.Seconds())

	result, err := s.db.Exec(`
		INSERT INTO sessions (token_hash, user_id, provider, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, tokenHash, userID, provider, expiresAt, now)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		TokenHash: tokenHash,
		UserID:    userID,
		Provider:  provider,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

func (s *SessionStore) GetByHash(tokenHash string) (*Session, error) {
	var sess Session
	err := s.db.QueryRow(`
		SELECT id, token_hash, user_id, provider, expires_at, created_at, last_used_at
		FROM sessions WHERE token_hash = $1
	`, tokenHash).Scan(&sess.ID, &sess.TokenHash, &sess.UserID, &sess.Provider,
		&sess.ExpiresAt, &sess.CreatedAt, &sess.LastUsedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	return &sess, err
}

func (s *SessionStore) Touch(id int64) {
	_, err := s.db.Exec(`
		UPDATE sessions SET last_used_at = strftime('%s', 'now') WHERE id = $1
	`, id)
	if err != nil {
		// Best-effort
	}
}

func (s *SessionStore) Delete(id int64) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = $1", id)
	return err
}

func (s *SessionStore) DeleteExpired() {
	now := time.Now().Unix()
	_, _ = s.db.Exec(`
		DELETE FROM sessions WHERE expires_at < $1
	`, now)
}
