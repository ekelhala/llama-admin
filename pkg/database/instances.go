package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"llama-admin/pkg/instance"
)

type InstanceStore struct {
	db *sql.DB
}

func NewInstanceStore(db *sql.DB) *InstanceStore {
	return &InstanceStore{db: db}
}

func (s *InstanceStore) Save(inst *instance.Instance) error {
	optsJSON, err := json.Marshal(inst.Opts)
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}

	ownerID := sql.NullInt64{}
	if inst.OwnerUserID != nil {
		ownerID = sql.NullInt64{Int64: *inst.OwnerUserID, Valid: true}
	}

	status := string(inst.Status())
	_, err = s.db.Exec(`
		INSERT INTO instances (id, name, status, created_at, updated_at, options_json, owner_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(name) DO UPDATE SET
			status = excluded.status,
			updated_at = excluded.updated_at,
			options_json = excluded.options_json,
			owner_user_id = excluded.owner_user_id
	`, inst.ID, inst.Name, status, inst.CreatedAt, inst.UpdatedAt, optsJSON, ownerID)

	return err
}

func (s *InstanceStore) LoadAll() ([]*instance.Instance, error) {
	rows, err := s.db.Query(`
		SELECT id, name, status, created_at, updated_at, options_json, owner_user_id
		FROM instances
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*instance.Instance
	for rows.Next() {
		var inst instance.Instance
		var statusStr string
		var optsJSON sql.NullString
		var ownerID sql.NullInt64

		if err := rows.Scan(&inst.ID, &inst.Name, &statusStr,
			&inst.CreatedAt, &inst.UpdatedAt, &optsJSON, &ownerID); err != nil {
			return nil, err
		}

		inst.RawStatus = instance.Status(statusStr)

		if optsJSON.Valid {
			inst.Opts = &instance.Options{}
			if err := json.Unmarshal([]byte(optsJSON.String), inst.Opts); err != nil {
				return nil, fmt.Errorf("unmarshal options for %s: %w", inst.Name, err)
			}
		}

		if ownerID.Valid {
			inst.OwnerUserID = &ownerID.Int64
		}

		instances = append(instances, &inst)
	}

	return instances, rows.Err()
}

func (s *InstanceStore) Get(name string) (*instance.Instance, error) {
	var inst instance.Instance
	var statusStr string
	var optsJSON sql.NullString
	var ownerID sql.NullInt64

	err := s.db.QueryRow(`
		SELECT id, name, status, created_at, updated_at, options_json, owner_user_id
		FROM instances WHERE name = $1
	`, name).Scan(&inst.ID, &inst.Name, &statusStr,
		&inst.CreatedAt, &inst.UpdatedAt, &optsJSON, &ownerID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance %q not found", name)
	}
	if err != nil {
		return nil, err
	}

	inst.RawStatus = instance.Status(statusStr)

	if optsJSON.Valid {
		inst.Opts = &instance.Options{}
		if err := json.Unmarshal([]byte(optsJSON.String), inst.Opts); err != nil {
			return nil, fmt.Errorf("unmarshal options: %w", err)
		}
	}

	if ownerID.Valid {
		inst.OwnerUserID = &ownerID.Int64
	}

	return &inst, nil
}

func (s *InstanceStore) Delete(name string) error {
	_, err := s.db.Exec("DELETE FROM instances WHERE name = $1", name)
	return err
}

func (s *InstanceStore) UpdateStatus(name string, status instance.Status) error {
	_, err := s.db.Exec(`
		UPDATE instances SET status = $1, updated_at = $2 WHERE name = $3
	`, status, time.Now().Unix(), name)
	return err
}
