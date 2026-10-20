// Package memory реализует хранилище данных в оперативной памяти.
package memory

import (
	"context"
	"sync"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// MemStorage реализует хранилище данных в оперативной памяти.
type MemStorage struct {
	mu        sync.RWMutex
	nextID    int
	secrets   map[string]models.CipherData
	nextLogID int
	changelog []models.Changelog
	user      map[string]models.LocalUser
	conflicts map[string]models.CipherData
}

// InitMemStorage создает новый экземпляр MemStorage.
func InitMemStorage() *MemStorage {
	return &MemStorage{
		nextID:    1,
		secrets:   make(map[string]models.CipherData),
		nextLogID: 1,
		changelog: make([]models.Changelog, 0),
		user:      make(map[string]models.LocalUser),
		conflicts: make(map[string]models.CipherData),
	}
}

// Ping проверяет доступность хранилища.
func (m *MemStorage) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
