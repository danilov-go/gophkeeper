// Package db реализует локальное хранилище данных на базе SQLite.
package db

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type storageDB struct {
	db *sql.DB
}

// NewStorageDB создает новый экземпляр storageDB.
func NewStorageDB(sql *sql.DB) *storageDB {
	return &storageDB{
		db: sql,
	}
}

// InitDB инициализирует подключение к базе данных SQLite.
func InitDB(path string) (*storageDB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &storageDB{db: db}
	if err := s.CreateTable(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	return s, nil
}

// СreateTable создает таблицы пользователей, секретов, журналов и конфликтов.
func (s *storageDB) CreateTable() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS device (
    		id TEXT PRIMARY KEY
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			login TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			salt TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			cipher BLOB NOT NULL,
			updated_at DATETIME NOT NULL,
			version INTEGER NOT NULL,
			deleted INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS changelog (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			secret_id TEXT NOT NULL,
			secret_data TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS conflicts (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			cipher BLOB NOT NULL,
			updated_at DATETIME NOT NULL,
			version INTEGER NOT NULL,
			deleted INTEGER NOT NULL
		);`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// GetDeviceID возвращает ID устройства.
func (s *storageDB) GetDeviceID(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var deviceID string
	query := `SELECT id FROM device LIMIT 1`
	err := s.db.QueryRowContext(ctx, query).Scan(&deviceID)
	if err == nil {
		return deviceID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id := uuid.New().String()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	query = `DELETE FROM device`
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, query); err != nil {
		return "", err
	}
	query = `INSERT INTO device (id) VALUES ($1)`
	if _, err = tx.ExecContext(ctx, query, id); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}

// Ping проверяет доступность хранилища.
func (s *storageDB) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close закрывает текущее соединение с базой данных.
func (s *storageDB) Close() error {
	return s.db.Close()
}
