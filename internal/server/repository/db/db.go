// Package db реализует хранилище данных в базе PostgreSQL.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Strategy interface {
	Rules(id int, version int, updatedAt time.Time, versions models.SecretVersions) models.SyncAction
}

type storageDB struct {
	db       *sql.DB
	strategy Strategy
}

// NewStorageDB создает новый экземпляр storageDB.
func NewStorageDB(sql *sql.DB, strategy Strategy) *storageDB {
	return &storageDB{
		db:       sql,
		strategy: strategy,
	}
}

// InitDB инициализирует подключение к базе данных PostgreSQL по переданной строке соединения.
func InitDB(ps string, strategy Strategy) (*storageDB, error) {
	db, err := sql.Open("pgx", ps)
	if err != nil {
		return nil, err
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		if closeErr := db.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	return &storageDB{
		db:       db,
		strategy: strategy,
	}, nil
}

func (s *storageDB) GetSecrets(ctx context.Context, userID int, v models.SecretVersions) ([]models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query := `SELECT id, data, version, updated_at FROM secrets WHERE user_id = $1;`
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cipherData []models.CipherData
	for rows.Next() {
		var id, version int
		var data []byte
		var updatedAt time.Time
		if err := rows.Scan(&id, &data, &version, &updatedAt); err != nil {
			return nil, err
		}
		if s.strategy.Rules(id, version, updatedAt, v) != models.ActionSkip {
			var secret models.CipherData
			if err := json.Unmarshal(data, &secret); err != nil {
				return nil, err
			}
			cipherData = append(cipherData, secret)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return cipherData, nil
}

func (s *storageDB) UpdateSecrets(ctx context.Context, userID int, changelogs models.Changelogs) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	query := `
		INSERT INTO secrets (id, user_id, type, data, version, deleted, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, id) DO UPDATE 
		SET type = EXCLUDED.type, 
		    data = EXCLUDED.data, 
		    version = EXCLUDED.version, 
		    deleted = EXCLUDED.deleted, 
		    updated_at = EXCLUDED.updated_at
		WHERE secrets.version < EXCLUDED.version;`
	now := time.Now()
	for _, changelog := range changelogs {
		secret := changelog.Secret
		id := changelog.SecretID
		secret.UpdatedAt = now
		cipherData, err := json.Marshal(secret)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, query, id, userID, secret.Type, cipherData, secret.Version, secret.Deleted, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *storageDB) SaveUser(ctx context.Context, login, hash, salt string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	query := `INSERT INTO users (login, hash, salt) VALUES ($1, $2, $3) RETURNING id;`
	var userID int
	err := s.db.QueryRowContext(ctx, query, login, hash, salt).Scan(&userID)
	if err != nil {
		return 0, models.ErrUserAlreadyExists
	}
	return userID, nil
}

func (s *storageDB) GetUser(ctx context.Context, login string) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}
	query := `SELECT id, login, hash, salt FROM users WHERE login = $1;`
	var user models.User
	err := s.db.QueryRowContext(ctx, query, login).Scan(&user.ID, &user.Login, &user.Hash, &user.Salt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, models.ErrUserNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (s *storageDB) Delete(ctx context.Context, userID, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	query := `
		UPDATE secrets 
		SET deleted = TRUE, version = version + 1, updated_at = $1 
		WHERE user_id = $2 AND id = $3;`
	_, err := s.db.ExecContext(ctx, query, time.Now(), userID, id)
	return err
}
