package db

import (
	"context"
	"database/sql"
	"errors"

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
	if err := s.createTable(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, errors.Join(err, closeErr)
		}
		return nil, err
	}
	return s, nil
}

func (s *storageDB) createTable() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			login TEXT PRIMARY KEY,
			hash TEXT NOT NULL,
			salt TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS secrets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			cipher BLOB NOT NULL,
			updated_at DATETIME NOT NULL,
			version INTEGER NOT NULL,
			deleted INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS changelog (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			action TEXT NOT NULL,
			secret_id INTEGER NOT NULL,
			secret_data TEXT NOT NULL
		);`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *storageDB) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *storageDB) Close() error {
	return s.db.Close()
}
