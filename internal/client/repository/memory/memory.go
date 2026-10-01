// Package memory реализует хранилище данных в оперативной памяти.
package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// MemStorage реализует хранилище данных в оперативной памяти.
type MemStorage struct {
	mu      sync.RWMutex
	nextID  int
	secrets map[int]models.CipherData
}

// InitMemStorage создает новый экземпляр MemStorage.
func InitMemStorage() *MemStorage {
	return &MemStorage{
		nextID:  1,
		secrets: make(map[int]models.CipherData),
	}
}

// SaveSecret сохраняет секрет.
func (m *MemStorage) SaveSecret(ctx context.Context, secret models.CipherData) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		m.secrets = make(map[int]models.CipherData)
	}
	id := m.nextID
	m.nextID++
	secret.ID = id
	m.secrets[id] = secret
	return id, nil
}

// GetSecret возвращает секрет по его ID.
func (m *MemStorage) GetSecret(ctx context.Context, id int) (models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return models.CipherData{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		return models.CipherData{}, errors.New("хранилище пустое")
	}
	if secret, ok := m.secrets[id]; ok {
		return secret, nil
	}
	return models.CipherData{}, errors.New("данные отсутствуют в хранилище")
}

// GetAllSecret возвращает список секретов одного типа.
func (m *MemStorage) GetAllSecret(ctx context.Context, sType models.SecretType) ([]models.CipherData, error) {
	var secrets []models.CipherData
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		return nil, errors.New("хранилище пустое")
	}
	for _, secret := range m.secrets {
		if secret.Type == sType {
			secrets = append(secrets, secret)
		}
	}
	if len(secrets) == 0 {
		return nil, errors.New("данные отсутствуют в хранилище")
	}
	return secrets, nil
}

// UpdateSecret обновляет существующие секрет.
func (m *MemStorage) UpdateSecret(ctx context.Context, secret models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		return errors.New("хранилище пустое")
	}
	if _, ok := m.secrets[secret.ID]; !ok {
		return errors.New("данные отсутствуют в хранилище")
	}
	m.secrets[secret.ID] = secret
	return nil
}

// DeleteSecret удаляет секрет по его ID.
func (m *MemStorage) DeleteSecret(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.secrets == nil {
		return errors.New("хранилище пустое")
	}
	if _, ok := m.secrets[id]; !ok {
		return errors.New("данные отсутствуют в хранилище")
	}
	delete(m.secrets, id)
	return nil
}

// Ping проверяет доступность хранилища.
func (m *MemStorage) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
