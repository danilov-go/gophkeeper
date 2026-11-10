package db_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const (
	expLogin = "login"
	expHash  = "hash"
	expSalt  = "salt"
)

var errTest = errors.New("test error")

func TestStorageDB_SaveUser(t *testing.T) {
	query := `INSERT OR REPLACE INTO users (login, hash, salt) VALUES (?, ?, ?)`
	tests := []struct {
		name          string
		login         string
		hash          string
		salt          string
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		wantErr       bool
		errText       string
	}{
		{
			name:  "положительный тест",
			login: expLogin,
			hash:  expHash,
			salt:  expSalt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(expLogin, expHash, expSalt).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:  "ошибка выполнения запроса",
			login: expLogin,
			hash:  expHash,
			salt:  expSalt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(expLogin, expHash, expSalt).
					WillReturnError(errTest)
			},
			wantErr: true,
			errText: "test error",
		},
		{
			name:          "ошибка отмены контекста",
			login:         expLogin,
			hash:          expHash,
			salt:          expSalt,
			cancelContext: true,
			setupMock:     func(mock sqlmock.Sqlmock) {},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err = storage.SaveUser(ctx, tt.login, tt.hash, tt.salt)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.Contains(t, err.Error(), tt.errText)
				}
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetUser(t *testing.T) {
	query := `SELECT hash, salt FROM users WHERE login = ?`
	tests := []struct {
		name          string
		login         string
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		wantUser      models.LocalUser
		wantErr       bool
		errText       string
	}{
		{
			name:  "положительный тест",
			login: expLogin,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"hash", "salt"}).
					AddRow(expHash, expSalt)
				mock.ExpectQuery(query).
					WithArgs(expLogin).
					WillReturnRows(rows)
			},
			wantUser: models.LocalUser{Hash: expHash, Salt: expSalt},
			wantErr:  false,
		},
		{
			name:  "пользователь отсутствует",
			login: "unknown_user",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs("unknown_user").
					WillReturnError(sql.ErrNoRows)
			},
			wantUser: models.LocalUser{},
			wantErr:  true,
			errText:  "пользователь отсутствует локально",
		},
		{
			name:  "ошибка базы данных",
			login: expLogin,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(expLogin).
					WillReturnError(errTest)
			},
			wantUser: models.LocalUser{},
			wantErr:  true,
			errText:  "test error",
		},
		{
			name:          "ошибка отмены контекста",
			login:         expLogin,
			cancelContext: true,
			setupMock:     func(mock sqlmock.Sqlmock) {},
			wantUser:      models.LocalUser{},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			user, err := storage.GetUser(ctx, tt.login)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.Contains(t, err.Error(), tt.errText)
				}
				assert.Equal(t, tt.wantUser, user)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantUser, user)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
