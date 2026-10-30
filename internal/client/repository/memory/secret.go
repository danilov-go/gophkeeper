package memory

import (
	"context"
	"errors"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/google/uuid"
)

// SaveSecret сохраняет секрет.
func (m *MemStorage) SaveSecret(ctx context.Context, secret models.CipherData) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		m.secrets = make(map[string]models.CipherData)
	}
	if secret.ID == "" {
		secret.ID = uuid.New().String()
	}
	secret.Version = 1
	if secret.UpdatedAt.IsZero() {
		secret.UpdatedAt = time.Now()
	}
	m.secrets[secret.ID] = secret
	m.changelog = append(m.changelog, models.Changelog{
		ID:       m.nextLogID,
		SecretID: secret.ID,
		Secret:   secret,
	})
	m.nextLogID++
	return secret.ID, nil
}

// GetSecret возвращает секрет по его ID.
func (m *MemStorage) GetSecret(ctx context.Context, id string) (models.CipherData, error) {
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
	localSecret, ok := m.secrets[secret.ID]
	if !ok {
		return errors.New("данные отсутствуют в хранилище")
	}
	secret.Version = localSecret.Version + 1
	secret.UpdatedAt = time.Now()
	m.secrets[secret.ID] = secret
	m.changelog = append(m.changelog, models.Changelog{
		ID:       m.nextLogID,
		SecretID: secret.ID,
		Secret:   secret,
	})
	m.nextLogID++
	return nil
}

// DeleteSecret удаляет секрет по его ID.
func (m *MemStorage) DeleteSecret(ctx context.Context, id string, sync bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		return errors.New("хранилище пустое")
	}
	secret, ok := m.secrets[id]
	if !ok {
		return errors.New("данные отсутствуют в хранилище")
	}
	if !sync {
		secret.Version++
		secret.Deleted = true
		m.changelog = append(m.changelog, models.Changelog{
			ID:       m.nextLogID,
			SecretID: secret.ID,
			Secret:   secret,
		})
		m.nextLogID++
	}
	delete(m.secrets, id)
	return nil
}
