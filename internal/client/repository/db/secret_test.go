package db_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const (
	secretID = "secretID"
)

func TestStorageDB_SaveSecret(t *testing.T) {
	querySecret := `INSERT INTO secrets (id, type, cipher, updated_at, version, deleted) VALUES (?, ?, ?, ?, ?, ?)`
	queryLog := `INSERT INTO changelog (secret_id, secret_data) VALUES (?, ?)`
	tests := []struct {
		name          string
		secret        models.CipherData
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		wantErr       bool
	}{
		{
			name: "положительный тест",
			secret: models.CipherData{
				ID:     secretID,
				Type:   models.TypeCard,
				Cipher: []byte("test"),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(querySecret).
					WithArgs(secretID, models.TypeCard, []byte("test"), sqlmock.AnyArg(), 1, 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(queryLog).
					WithArgs(secretID, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "положительный тест (генерация ID)",
			secret: models.CipherData{
				ID:     "",
				Type:   models.TypeLoginPassword,
				Cipher: []byte("test"),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(querySecret).
					WithArgs(sqlmock.AnyArg(), models.TypeLoginPassword, []byte("test"), sqlmock.AnyArg(), 1, 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(queryLog).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "ошибка базы данных",
			secret: models.CipherData{
				ID: secretID,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(querySecret).
					WithArgs(secretID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errTest)
				mock.ExpectRollback()
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
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			id, err := storage.SaveSecret(ctx, tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, id)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetSecret(t *testing.T) {
	query := `SELECT id, type, cipher, updated_at, version, deleted FROM secrets WHERE id = ? AND deleted = 0`
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name      string
		id        string
		setupMock func(mock sqlmock.Sqlmock)
		want      models.CipherData
		wantErr   bool
		errText   string
	}{
		{
			name: "положительный тест",
			id:   secretID,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "type", "cipher", "updated_at", "version", "deleted"}).
					AddRow(secretID, models.TypeCard, []byte("test"), now, 1, 0)
				mock.ExpectQuery(query).WithArgs(secretID).WillReturnRows(rows)
			},
			want:    models.CipherData{ID: secretID, Type: models.TypeCard, Cipher: []byte("test"), UpdatedAt: now, Version: 1, Deleted: false},
			wantErr: false,
		},
		{
			name: "данные отсутствуют в хранилище",
			id:   "unknow",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WithArgs("unknow").WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errText: "данные отсутствуют в хранилище",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			secret, err := storage.GetSecret(t.Context(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.Contains(t, err.Error(), tt.errText)
				}
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, secret)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetAllSecret(t *testing.T) {
	query := `SELECT id, type, cipher, updated_at, version, deleted FROM secrets WHERE type = ? AND deleted = 0`
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name      string
		sType     models.SecretType
		setupMock func(mock sqlmock.Sqlmock)
		wantCount int
		wantErr   bool
		errText   string
	}{
		{
			name:  "положительный тест",
			sType: models.TypeBinary,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "type", "cipher", "updated_at", "version", "deleted"}).
					AddRow(secretID1, models.TypeBinary, []byte("secret1"), now, 2, 0).
					AddRow(secretID2, models.TypeBinary, []byte("secret2"), now, 1, 0)
				mock.ExpectQuery(query).WithArgs(models.TypeBinary).WillReturnRows(rows)
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "пустой список",
			sType: models.TypeCard,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "type", "cipher", "updated_at", "version", "deleted"})
				mock.ExpectQuery(query).WithArgs(models.TypeCard).WillReturnRows(rows)
			},
			wantErr: true,
			errText: "данные отсутствуют в хранилище",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			secrets, err := storage.GetAllSecret(t.Context(), tt.sType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, secrets, tt.wantCount)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_UpdateSecret(t *testing.T) {
	querySelect := `SELECT version FROM secrets WHERE id = ?`
	queryUpdate := `UPDATE secrets SET version = ?, cipher = ?, updated_at = ? WHERE id = ?`
	queryLog := `INSERT INTO changelog (secret_id, secret_data) VALUES (?, ?)`
	secret := models.CipherData{
		ID:     secretID,
		Cipher: []byte("test"),
	}
	tests := []struct {
		name      string
		secret    models.CipherData
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
		errText   string
	}{
		{
			name:   "положительный тест",
			secret: secret,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"version"}).AddRow(3)
				mock.ExpectQuery(querySelect).WithArgs(secretID).WillReturnRows(rows)
				mock.ExpectExec(queryUpdate).WithArgs(4, []byte("test"), sqlmock.AnyArg(), secretID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(queryLog).WithArgs(secretID, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:   "обновление несуществующего секрета",
			secret: secret,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(querySelect).WithArgs(secretID).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: true,
			errText: "данные отсутствуют в хранилище",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			err = storage.UpdateSecret(t.Context(), tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_DeleteSecret(t *testing.T) {
	queryDelete := `DELETE FROM secrets WHERE id = ?`
	querySelect := `SELECT id, type, cipher, version FROM secrets WHERE id = ?`
	queryLog := `INSERT INTO changelog (secret_id, secret_data) VALUES (?, ?)`
	tests := []struct {
		name      string
		id        string
		sync      bool
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
		errText   string
	}{
		{
			name: "положительный тест (sync = true)",
			id:   secretID,
			sync: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(queryDelete).WithArgs(secretID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "положительный тест (sync = false)",
			id:   secretID,
			sync: false,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"id", "type", "cipher", "version"}).
					AddRow(secretID, models.TypeCard, []byte("data"), 5)
				mock.ExpectQuery(querySelect).WithArgs(secretID).WillReturnRows(rows)
				mock.ExpectExec(queryLog).WithArgs(secretID, sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(queryDelete).WithArgs(secretID).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name: "удаление несуществующего секрета",
			id:   secretID,
			sync: false,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(querySelect).WithArgs(secretID).WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: true,
			errText: "данные отсутствуют в хранилище",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			assert.NoError(t, err)
			defer func() { _ = sql.Close() }()
			tt.setupMock(mock)
			storage := db.NewStorageDB(sql)
			err = storage.DeleteSecret(t.Context(), tt.id, tt.sync)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
