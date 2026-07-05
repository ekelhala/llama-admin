package database

import (
	"database/sql"
	"time"
)

type AllowedEmailStore struct {
	db *sql.DB
}

func NewAllowedEmailStore(db *sql.DB) *AllowedEmailStore {
	return &AllowedEmailStore{db: db}
}

type AllowedEmail struct {
	Email       string  `json:"email"`
	Source      string  `json:"source"` // "config" or "api"
	AddedByUserID *int64 `json:"added_by_user_id,omitempty"`
}

func (s *AllowedEmailStore) Add(email string, addedByUserID *int64) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`
		INSERT INTO allowed_emails (email, added_by_user_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT(email) DO NOTHING
	`, email, addedByUserID, now)
	return err
}

func (s *AllowedEmailStore) Remove(email string) error {
	_, err := s.db.Exec("DELETE FROM allowed_emails WHERE email = $1", email)
	return err
}

func (s *AllowedEmailStore) List() ([]AllowedEmail, error) {
	rows, err := s.db.Query(`
		SELECT email, added_by_user_id, created_at
		FROM allowed_emails
		ORDER BY email
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []AllowedEmail
	for rows.Next() {
		var ae AllowedEmail
		var createdAt int64
		if err := rows.Scan(&ae.Email, &ae.AddedByUserID, &createdAt); err != nil {
			return nil, err
		}
		ae.Source = "api"
		emails = append(emails, ae)
	}
	return emails, rows.Err()
}

func (s *AllowedEmailStore) Contains(email string) bool {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM allowed_emails WHERE email = $1
	`, email).Scan(&count)
	return err == nil && count > 0
}
