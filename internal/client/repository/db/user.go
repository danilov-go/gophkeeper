package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (s *storageDB) SaveUser(ctx context.Context, login, hash, salt string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	query := `INSERT OR REPLACE INTO users (login, hash, salt) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, login, hash, salt)
	return err
}

func (s *storageDB) GetUser(ctx context.Context, login string) (models.LocalUser, error) {
	var user models.LocalUser
	if err := ctx.Err(); err != nil {
		return user, err
	}
	query := `SELECT hash, salt FROM users WHERE login = ?`
	err := s.db.QueryRowContext(ctx, query, login).Scan(&user.Hash, &user.Salt)
	if errors.Is(err, sql.ErrNoRows) {
		return user, errors.New("пользователь отсутствует локально")
	}
	return user, err
}
