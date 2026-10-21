package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/server/repository/db"
	"github.com/danilov-go/gophkeeper/internal/strategy"
	"github.com/stretchr/testify/assert"
)

const (
	expLogin  = "login"
	expHash   = "password"
	expSalt   = "test_salt"
	userID    = 15
	secretID1 = "test1"
	secretID2 = "test2"
)

var errTest = errors.New("test error")

func TestStorageDB_SaveUser(t *testing.T) {
	query := `INSERT INTO users (login, hash, salt) VALUES ($1, $2, $3) RETURNING id;`
	tests := []struct {
		name      string
		login     string
		hash      string
		salt      string
		setupMock func(mock sqlmock.Sqlmock)
		wantID    int
		wantErr   error
	}{
		{
			name:  "положительный тест",
			login: expLogin,
			hash:  expHash,
			salt:  expSalt,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(userID)
				mock.ExpectQuery(query).WithArgs(expLogin, expHash, expSalt).WillReturnRows(rows)
			},
			wantID:  userID,
			wantErr: nil,
		},
		{
			name:  "пользователь уже существует",
			login: expLogin,
			hash:  expHash,
			salt:  expSalt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(expLogin, expHash, expSalt).WillReturnError(errTest)
			},
			wantID:  0,
			wantErr: models.ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			strategy := strategy.NewClientStrategy()
			storage := db.NewStorageDB(sql, strategy)
			id, err := storage.SaveUser(context.Background(), tt.login, tt.hash, tt.salt)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantID, id)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetUser(t *testing.T) {
	query := `SELECT id, login, hash, salt FROM users WHERE login = $1;`
	tests := []struct {
		name      string
		login     string
		setupMock func(mock sqlmock.Sqlmock)
		wantUser  models.User
		wantErr   error
	}{
		{
			name:  "положительный тест",
			login: expLogin,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "login", "hash", "salt"}).
					AddRow(userID, expLogin, expHash, expSalt)
				mock.ExpectQuery(query).WithArgs(expLogin).WillReturnRows(rows)
			},
			wantUser: models.User{ID: userID, Login: expLogin, Hash: expHash, Salt: expSalt},
			wantErr:  nil,
		},
		{
			name:  "пользователь не найден",
			login: "unknown",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs("unknown").WillReturnError(sql.ErrNoRows)
			},
			wantUser: models.User{},
			wantErr:  models.ErrUserNotFound,
		},
		{
			name:  "ошибка базы данных",
			login: expLogin,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(expLogin).WillReturnError(errTest)
			},
			wantUser: models.User{},
			wantErr:  errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			strategy := strategy.NewClientStrategy()
			storage := db.NewStorageDB(sql, strategy)
			user, err := storage.GetUser(context.Background(), tt.login)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantUser, user)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetSecrets(t *testing.T) {
	query := `SELECT id, data, version, updated_at FROM secrets WHERE user_id = $1;`
	now := time.Now().Truncate(time.Millisecond)
	secret := models.CipherData{ID: secretID1, Version: 1, UpdatedAt: now}
	secretBytes, err := json.Marshal(secret)
	assert.NoError(t, err)
	tests := []struct {
		name        string
		userID      int
		setupMock   func(mock sqlmock.Sqlmock)
		wantSecrets []models.CipherData
		wantErr     error
	}{
		{
			name:   "положительный тест",
			userID: userID,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "data", "version", "updated_at"}).
					AddRow(secretID1, secretBytes, 1, now)
				mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
			},
			wantSecrets: []models.CipherData{secret},
			wantErr:     nil,
		},
		{
			name:   "секреты отсутствуют",
			userID: userID,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "data", "version", "updated_at"})
				mock.ExpectQuery(query).WithArgs(userID).WillReturnRows(rows)
			},
			wantSecrets: nil,
			wantErr:     nil,
		},
		{
			name:   "ошибка БД",
			userID: userID,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs(userID).WillReturnError(errTest)
			},
			wantSecrets: nil,
			wantErr:     errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			strategy := strategy.NewClientStrategy()
			storage := db.NewStorageDB(sql, strategy)
			res, err := storage.GetSecrets(context.Background(), tt.userID, models.SecretVersions{})
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantSecrets, res)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_UpdateSecrets(t *testing.T) {
	query := `
	INSERT INTO secrets (id, user_id, type, data, version, deleted, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (user_id, id) DO UPDATE
	SET type = EXCLUDED.type, data = EXCLUDED.data, version = EXCLUDED.version, deleted = EXCLUDED.deleted, updated_at = EXCLUDED.updated_at
	WHERE secrets.version < EXCLUDED.version;`
	now := time.Now().Truncate(time.Millisecond)
	secret := models.CipherData{ID: secretID1, Type: models.TypeCard, Version: 2, Deleted: false, UpdatedAt: now}
	secretBytes, err := json.Marshal(secret)
	assert.NoError(t, err)
	tests := []struct {
		name       string
		userID     int
		changelogs models.Changelogs
		setupMock  func(mock sqlmock.Sqlmock)
		wantErr    error
	}{
		{
			name:   "положительный тест",
			userID: userID,
			changelogs: models.Changelogs{
				{SecretID: secretID1, Secret: secret},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(query).
					WithArgs(secretID1, userID, models.TypeCard, secretBytes, 2, false, now).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:   "ошибка БД",
			userID: userID,
			changelogs: models.Changelogs{
				{SecretID: secretID1, Secret: secret},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(query).
					WithArgs(secretID1, userID, models.TypeCard, secretBytes, 2, false, now).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			strategy := strategy.NewClientStrategy()
			storage := db.NewStorageDB(sql, strategy)
			err = storage.UpdateSecrets(context.Background(), tt.userID, tt.changelogs)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_Delete(t *testing.T) {
	query := `
	UPDATE secrets
	SET deleted = TRUE, version = version + 1, updated_at = $1
	WHERE user_id = $2 AND id = $3;`
	tests := []struct {
		name          string
		userID        int
		secretID      string
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		wantErr       error
	}{
		{
			name:     "успешное удаление секрета",
			userID:   userID,
			secretID: secretID1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(sqlmock.AnyArg(), userID, secretID1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: nil,
		},
		{
			name:     "секрет для удаления не найден",
			userID:   userID,
			secretID: secretID2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(sqlmock.AnyArg(), userID, secretID2).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: nil,
		},
		{
			name:     "ошибка БД",
			userID:   userID,
			secretID: secretID1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(sqlmock.AnyArg(), userID, secretID1).
					WillReturnError(errTest)
			},
			wantErr: errTest,
		},
		{
			name:          "ошибка отмененного контекста",
			userID:        userID,
			secretID:      secretID1,
			cancelContext: true,
			setupMock:     func(mock sqlmock.Sqlmock) {},
			wantErr:       context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			strategy := strategy.NewClientStrategy()
			storage := db.NewStorageDB(sql, strategy)
			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err = storage.Delete(ctx, tt.userID, tt.secretID)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
