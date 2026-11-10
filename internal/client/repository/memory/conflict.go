package memory

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// SaveConflict сохраняет или обновляет информацию о конфликтующем секрете.
func (m *MemStorage) SaveConflict(ctx context.Context, secret models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conflicts[secret.ID] = secret
	return nil
}

// GetConflicts возвращает все конфликтные записи.
func (m *MemStorage) GetConflicts(ctx context.Context) ([]models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var secrets []models.CipherData
	for _, secret := range m.conflicts {
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

// DeleteConflict удаляет конфликтную запись.
func (m *MemStorage) DeleteConflict(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conflicts, id)
	return nil
}
