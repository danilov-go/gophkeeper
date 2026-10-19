package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (s *storageDB) GetVersion(ctx context.Context) (map[int]models.VersionTime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query := `SELECT id, version, updated_at FROM secrets`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versions := make(map[int]models.VersionTime)
	for rows.Next() {
		var id, version int
		var updatedAt time.Time
		if err := rows.Scan(&id, &version, &updatedAt); err != nil {
			return nil, err
		}
		versions[id] = models.VersionTime{
			Version:   version,
			UpdatedAt: updatedAt,
		}
	}
	return versions, nil
}

func (s *storageDB) GetChangelog() ([]models.Changelog, error) {
	query := `SELECT id, action, secret_id, secret_data FROM changelog`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var changelog []models.Changelog
	for rows.Next() {
		var log models.Changelog
		var secretDataStr string
		if err := rows.Scan(&log.ID, &log.Action, &log.SecretID, &secretDataStr); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(secretDataStr), &log.Secret); err != nil {
			return nil, err
		}
		changelog = append(changelog, log)
	}
	return changelog, nil
}

func (s *storageDB) DeleteChangelog(id int) error {
	query := `DELETE FROM changelog WHERE id <= ?`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *storageDB) SyncUpdate(ctx context.Context, cipherData []models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	query := `INSERT OR REPLACE INTO secrets (id, type, cipher, updated_at, version, deleted) 
			 VALUES (?, ?, ?, ?, ?, ?)`
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, cipher := range cipherData {
		deletedInt := 0
		if cipher.Deleted {
			deletedInt = 1
		}
		_, err = tx.ExecContext(ctx, query, cipher.ID, cipher.Type, cipher.Cipher, cipher.UpdatedAt, cipher.Version, deletedInt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
