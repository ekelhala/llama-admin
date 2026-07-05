package database

import (
	"database/sql"
	"fmt"
	"time"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

type User struct {
	ID             int64     `json:"id"`
	Provider       string    `json:"provider"`
	ProviderUserID string    `json:"provider_user_id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	AvatarURL      string    `json:"avatar_url"`
	CreatedAt      int64     `json:"created_at"`
	UpdatedAt      int64     `json:"updated_at"`
}

func (s *UserStore) Upsert(provider, providerUserID, username, email, avatarURL string) (*User, error) {
	now := time.Now().Unix()
	var user User
	err := s.db.QueryRow(`
		INSERT INTO users (provider, provider_user_id, username, email, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(provider, provider_user_id) DO UPDATE SET
			username = excluded.username,
			email = excluded.email,
			avatar_url = excluded.avatar_url,
			updated_at = excluded.updated_at
		RETURNING id, provider, provider_user_id, username, email, avatar_url, created_at, updated_at
	`, provider, providerUserID, username, email, avatarURL, now, now).Scan(
		&user.ID, &user.Provider, &user.ProviderUserID, &user.Username,
		&user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return &user, nil
}

func (s *UserStore) GetByID(id int64) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, provider, provider_user_id, username, email, avatar_url, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Provider, &user.ProviderUserID, &user.Username,
		&user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return &user, err
}

func (s *UserStore) GetByProvider(provider, providerUserID string) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, provider, provider_user_id, username, email, avatar_url, created_at, updated_at
		FROM users WHERE provider = $1 AND provider_user_id = $2
	`, provider, providerUserID).Scan(&user.ID, &user.Provider, &user.ProviderUserID, &user.Username,
		&user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found for provider %s / %s", provider, providerUserID)
	}
	return &user, err
}

func GetUserByID(db *sql.DB, id int64) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT id, provider, provider_user_id, username, email, avatar_url, created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Provider, &user.ProviderUserID, &user.Username,
		&user.Email, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return &user, err
}
