package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/google/uuid"
)

func (s *storageDB) SaveSecret(ctx context.Context, secret models.CipherData) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if secret.ID == "" {
		secret.ID = uuid.New().String()
	}
	secret.Version = 1
	if secret.UpdatedAt.IsZero() {
		secret.UpdatedAt = time.Now()
	}
	query := `INSERT INTO secrets (id, type, cipher, updated_at, version, deleted) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = tx.ExecContext(ctx, query, secret.ID, secret.Type, secret.Cipher, secret.UpdatedAt, secret.Version, 0)
	if err != nil {
		return "", err
	}
	cipherData, err := json.Marshal(secret)
	if err != nil {
		return "", err
	}
	query = `INSERT INTO changelog (action, secret_id, secret_data) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, query, models.ActionCreate, secret.ID, string(cipherData))
	if err != nil {
		return "", err
	}
	return secret.ID, tx.Commit()
}

func (s *storageDB) GetSecret(ctx context.Context, id string) (models.CipherData, error) {
	var secret models.CipherData
	if err := ctx.Err(); err != nil {
		return secret, err
	}
	var deletedInt int
	query := `SELECT id, type, cipher, updated_at, version, deleted FROM secrets WHERE id = ? AND deleted = 0`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&secret.ID, &secret.Type, &secret.Cipher, &secret.UpdatedAt, &secret.Version, &deletedInt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.CipherData{}, errors.New("данные отсутствуют в хранилище")
	}
	if err != nil {
		return models.CipherData{}, err
	}
	secret.Deleted = deletedInt == 1
	return secret, nil
}

func (s *storageDB) GetAllSecret(ctx context.Context, sType models.SecretType) ([]models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query := `SELECT id, type, cipher, updated_at, version, deleted FROM secrets WHERE type = ? AND deleted = 0`
	rows, err := s.db.QueryContext(ctx, query, sType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var secrets []models.CipherData
	for rows.Next() {
		var secret models.CipherData
		var deletedInt int
		if err := rows.Scan(&secret.ID, &secret.Type, &secret.Cipher, &secret.UpdatedAt, &secret.Version, &deletedInt); err != nil {
			return nil, err
		}
		secret.Deleted = deletedInt == 1
		secrets = append(secrets, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return nil, errors.New("данные отсутствуют в хранилище")
	}
	return secrets, nil
}

func (s *storageDB) UpdateSecret(ctx context.Context, secret models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	query := `SELECT version FROM secrets WHERE id = ?`
	err = tx.QueryRowContext(ctx, query, secret.ID).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("данные отсутствуют в хранилище")
	}
	if err != nil {
		return err
	}
	secret.Version = version + 1
	secret.UpdatedAt = time.Now()
	query = `UPDATE secrets SET version = ?, cipher = ?, updated_at = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, secret.Version, secret.Cipher, secret.UpdatedAt, secret.ID)
	if err != nil {
		return err
	}
	cipherData, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	query = `INSERT INTO changelog (action, secret_id, secret_data) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, query, models.ActionUpdate, secret.ID, string(cipherData))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *storageDB) DeleteSecret(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var secret models.CipherData
	query := `SELECT id, type, cipher, version FROM secrets WHERE id = ? AND deleted = 0`
	err = tx.QueryRowContext(ctx, query, id).
		Scan(&secret.ID, &secret.Type, &secret.Cipher, &secret.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("данные отсутствуют в хранилище")
	}
	if err != nil {
		return err
	}
	secret.Version++
	secret.Deleted = true
	secret.UpdatedAt = time.Now()
	query = `UPDATE secrets SET version = ?, deleted = 1, updated_at = ? WHERE id = ?`
	_, err = tx.ExecContext(ctx, query, secret.Version, secret.UpdatedAt, id)
	if err != nil {
		return err
	}
	cipherData, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	query = `INSERT INTO changelog (action, secret_id, secret_data) VALUES (?, ?, ?)`
	_, err = tx.ExecContext(ctx, query, models.ActionDelete, secret.ID, string(cipherData))
	if err != nil {
		return err
	}
	return tx.Commit()
}
