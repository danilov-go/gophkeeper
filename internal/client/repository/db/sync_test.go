package db_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const (
	secretID1 = "secretID1"
	secretID2 = "secretID12"
)

func TestStorageDB_GetVersion(t *testing.T) {
	query := `SELECT id, version, updated_at FROM secrets`
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name          string
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		want          map[string]models.VersionTime
		wantErr       bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "version", "updated_at"}).
					AddRow(secretID1, 1, now).
					AddRow(secretID2, 5, now)
				mock.ExpectQuery(query).WillReturnRows(rows)
			},
			want: map[string]models.VersionTime{
				secretID1: {Version: 1, UpdatedAt: now},
				secretID2: {Version: 5, UpdatedAt: now},
			},
			wantErr: false,
		},
		{
			name: "ошибка выполнения запроса",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WillReturnError(errTest)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:          "ошибка отмены контекста",
			cancelContext: true,
			setupMock:     func(mock sqlmock.Sqlmock) {},
			want:          nil,
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
			version, err := storage.GetVersion(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, version)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetChangelog(t *testing.T) {
	query := `SELECT id, secret_id, secret_data FROM changelog`
	secret := models.CipherData{ID: secretID1, Version: 2, Cipher: []byte("test")}
	secretBytes, err := json.Marshal(secret)
	assert.NoError(t, err)
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		want      []models.Changelog
		wantErr   bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "secret_id", "secret_data"}).
					AddRow(1, secretID1, string(secretBytes))
				mock.ExpectQuery(query).WillReturnRows(rows)
			},
			want: []models.Changelog{
				{ID: 1, SecretID: secretID1, Secret: secret},
			},
			wantErr: false,
		},
		{
			name: "ошибка базы данных",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WillReturnError(errTest)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			changelog, err := storage.GetChangelog()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, changelog)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_DeleteChangelog(t *testing.T) {
	query := `DELETE FROM changelog WHERE id <= ?`
	id := 15
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 5))
			},
			wantErr: false,
		},
		{
			name: "ошибка удаления",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(id).WillReturnError(errTest)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			err = storage.DeleteChangelog(id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_SyncUpdate(t *testing.T) {
	query := `INSERT OR REPLACE INTO secrets (id, type, cipher, updated_at, version, deleted) VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().Truncate(time.Second)
	secrets := []models.CipherData{
		{ID: secretID1, Type: models.TypeCard, Cipher: []byte("test"), UpdatedAt: now, Version: 2, Deleted: false},
		{ID: secretID2, Type: models.TypeBinary, Cipher: []byte("test"), UpdatedAt: now, Version: 1, Deleted: true},
	}
	tests := []struct {
		name          string
		cipherData    []models.CipherData
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		wantErr       bool
	}{
		{
			name:       "положительный тест",
			cipherData: secrets,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(query).WithArgs(secretID1, models.TypeCard, []byte("test"), now, 2, 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(query).WithArgs(secretID2, models.TypeBinary, []byte("test"), now, 1, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:       "ошибка внутри транзакции",
			cipherData: secrets,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(query).WithArgs(secretID1, models.TypeCard, []byte("test"), now, 2, 0).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: true,
		},
		{
			name:          "ошибка отмены контекста",
			cipherData:    secrets,
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
			err = storage.SyncUpdate(ctx, tt.cipherData)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
