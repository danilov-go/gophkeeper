package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestStorageDB_SaveConflict(t *testing.T) {
	query := `INSERT OR REPLACE INTO conflicts (id, type, cipher, updated_at, version, deleted) VALUES (?, ?, ?, ?, ?, ?)`
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name          string
		secret        models.CipherData
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		wantErr       bool
	}{
		{
			name: "положительный тест (Deleted=false)",
			secret: models.CipherData{
				ID:        secretID1,
				Type:      models.TypeCard,
				Cipher:    []byte("test"),
				UpdatedAt: now,
				Version:   2,
				Deleted:   false,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(secretID1, models.TypeCard, []byte("test"), now, 2, 0).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "положительный тест - (Deleted=true)",
			secret: models.CipherData{
				ID:        secretID2,
				Type:      models.TypeBinary,
				Cipher:    []byte("test"),
				UpdatedAt: now,
				Version:   1,
				Deleted:   true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(secretID2, models.TypeBinary, []byte("test"), now, 1, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:   "ошибка выполнения запроса",
			secret: models.CipherData{ID: secretID1},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(secretID1, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(errTest)
			},
			wantErr: true,
		},
		{
			name:          "ошибка отмены контекста",
			secret:        models.CipherData{ID: secretID1},
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
			err = storage.SaveConflict(ctx, tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_GetConflicts(t *testing.T) {
	query := `SELECT id, type, cipher, updated_at, version, deleted FROM conflicts`
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name          string
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		want          []models.CipherData
		wantErr       bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "type", "cipher", "updated_at", "version", "deleted"}).
					AddRow(secretID1, models.TypeCard, []byte("secret1"), now, 2, 0).
					AddRow(secretID2, models.TypeBinary, []byte("secret2"), now, 1, 1)
				mock.ExpectQuery(query).WillReturnRows(rows)
			},
			want: []models.CipherData{
				{ID: secretID1, Type: models.TypeCard, Cipher: []byte("secret1"), UpdatedAt: now, Version: 2, Deleted: false},
				{ID: secretID2, Type: models.TypeBinary, Cipher: []byte("secret2"), UpdatedAt: now, Version: 1, Deleted: true},
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
			conflicts, err := storage.GetConflicts(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, conflicts)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestStorageDB_DeleteConflict(t *testing.T) {
	query := `DELETE FROM conflicts WHERE id = ?`
	tests := []struct {
		name      string
		id        string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "положительный тест",
			id:   secretID1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(secretID1).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "ошибка при удалении",
			id:   secretID1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).WithArgs(secretID1).WillReturnError(errTest)
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
			err = storage.DeleteConflict(context.Background(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
