package db_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/stretchr/testify/assert"
)

func TestNewStorageDB(t *testing.T) {
	sql, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = sql.Close() }()
	storage := db.NewStorageDB(sql)
	assert.NotNil(t, storage)
}

func TestStorageDB_InitDB(t *testing.T) {
	storage, err := db.InitDB(":memory:")
	if assert.NoError(t, err) && assert.NotNil(t, storage) {
		err = storage.Ping(context.Background())
		assert.NoError(t, err)
		assert.NoError(t, storage.Close())
	}
}

func TestStorageDB_СreateTable(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS users (
			login TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			salt TEXT NOT NULL
		);`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			cipher BLOB NOT NULL,
			updated_at DATETIME NOT NULL,
			version INTEGER NOT NULL,
			deleted INTEGER NOT NULL
		);`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS changelog (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			secret_id TEXT NOT NULL,
			secret_data TEXT NOT NULL
		);`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS conflicts (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			cipher BLOB NOT NULL,
			updated_at DATETIME NOT NULL,
			version INTEGER NOT NULL,
			deleted INTEGER NOT NULL
		);`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "ошибка при создании таблицы",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS users (
			login TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			salt TEXT NOT NULL
		);`).WillReturnError(errTest)
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
			err = storage.СreateTable()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}
