package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/danilov-go/gophkeeper/internal/client/repository/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	deviceID = "deviceID"
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

func TestStorageDB_CreateTable(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`CREATE TABLE IF NOT EXISTS device (
    		id TEXT PRIMARY KEY
		);`).WillReturnResult(sqlmock.NewResult(0, 0))
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
			err = storage.CreateTable()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestStorageDB_GetDeviceID(t *testing.T) {
	query := `SELECT id FROM device LIMIT 1`
	deleteQuery := `DELETE FROM device`
	insertQuery := `INSERT INTO device (id) VALUES ($1)`
	tests := []struct {
		name          string
		cancelContext bool
		setupMock     func(mock sqlmock.Sqlmock)
		id            string
		generated     bool
		wantErr       bool
		expErr        string
	}{
		{
			name: "положительный тест",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(deviceID)
				mock.ExpectQuery(query).WillReturnRows(rows)
			},
			id:        deviceID,
			generated: false,
			wantErr:   false,
		},
		{
			name: "положительный тест (генерация ID)",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WillReturnError(sql.ErrNoRows)
				mock.ExpectBegin()
				mock.ExpectExec(deleteQuery).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(insertQuery).
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
			generated: true,
			wantErr:   false,
		},
		{
			name: "ошибка базы данных",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).WillReturnError(sql.ErrNoRows)
				mock.ExpectBegin()
				mock.ExpectExec(deleteQuery).WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantErr: true,
			expErr:  errTest.Error(),
		},
		{
			name:          "ошибка отмены контекста",
			cancelContext: true,
			setupMock:     func(mock sqlmock.Sqlmock) {},
			wantErr:       true,
			expErr:        context.Canceled.Error(),
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
			id, err := storage.GetDeviceID(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expErr != "" {
					assert.Contains(t, err.Error(), tt.expErr)
				}
				assert.Empty(t, id)
				return
			}
			assert.NoError(t, err)
			if tt.generated {
				_, parseErr := uuid.Parse(id)
				assert.NoError(t, parseErr)
			} else {
				assert.Equal(t, tt.id, id)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
