package database

import (
	"database/sql"
	"fmt"
)

type PermissionStore struct {
	db *sql.DB
}

func NewPermissionStore(db *sql.DB) *PermissionStore {
	return &PermissionStore{db: db}
}

type KeyPermission struct {
	KeyID       int64  `json:"key_id"`
	InstanceID  int64  `json:"instance_id"`
	InstanceName string `json:"instance_name,omitempty"`
}

func (s *PermissionStore) Grant(keyID, instanceID int64) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO key_permissions (key_id, instance_id) VALUES ($1, $2)
	`, keyID, instanceID)
	return err
}

func (s *PermissionStore) Revoke(keyID, instanceID int64) error {
	_, err := s.db.Exec(`
		DELETE FROM key_permissions WHERE key_id = $1 AND instance_id = $2
	`, keyID, instanceID)
	return err
}

func (s *PermissionStore) List(keyID int64) ([]KeyPermission, error) {
	rows, err := s.db.Query(`
		SELECT kp.key_id, kp.instance_id, i.name
		FROM key_permissions kp
		LEFT JOIN instances i ON kp.instance_id = i.id
		WHERE kp.key_id = $1
	`, keyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []KeyPermission
	for rows.Next() {
		var p KeyPermission
		if err := rows.Scan(&p.KeyID, &p.InstanceID, &p.InstanceName); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (s *PermissionStore) HasPermission(keyID, instanceID int64) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM key_permissions WHERE key_id = $1 AND instance_id = $2
	`, keyID, instanceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PermissionStore) InstanceExists(instanceID int64) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM instances WHERE id = $1
	`, instanceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *PermissionStore) GetInstanceName(instanceID int64) (string, error) {
	var name string
	err := s.db.QueryRow(`
		SELECT name FROM instances WHERE id = $1
	`, instanceID).Scan(&name)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("instance %d not found", instanceID)
	}
	return name, err
}
