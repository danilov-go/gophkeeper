package db

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (s *storageDB) SaveConflict(ctx context.Context, secret models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	query := `INSERT OR REPLACE INTO secrets_conflicts (id, type, cipher, updated_at, version, deleted) VALUES (?, ?, ?, ?, ?, ?)`
	deletedInt := 0
	if secret.Deleted {
		deletedInt = 1
	}
	_, err := s.db.ExecContext(ctx, query, secret.ID, secret.Type, secret.Cipher, secret.UpdatedAt, secret.Version, deletedInt)
	return err
}

func (s *storageDB) GetConflicts(ctx context.Context) ([]models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query := `SELECT id, type, cipher, updated_at, version, deleted FROM secrets_conflicts`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conflicts []models.CipherData
	for rows.Next() {
		var cd models.CipherData
		var deletedInt int
		if err := rows.Scan(&cd.ID, &cd.Type, &cd.Cipher, &cd.UpdatedAt, &cd.Version, &deletedInt); err != nil {
			return nil, err
		}
		cd.Deleted = deletedInt == 1
		conflicts = append(conflicts, cd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return conflicts, nil
}

func (s *storageDB) DeleteConflict(ctx context.Context, id int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM secrets_conflicts WHERE id = ?`, id)
	return err
}
