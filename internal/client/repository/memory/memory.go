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
	mu        sync.RWMutex
	nextID    int
	secrets   map[int]models.CipherData
	nextLogID int
	changelog []models.Changelog
}

// InitMemStorage создает новый экземпляр MemStorage.
func InitMemStorage() *MemStorage {
	return &MemStorage{
		nextID:    1,
		secrets:   make(map[int]models.CipherData),
		nextLogID: 1,
		changelog: make([]models.Changelog, 0),
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
	secret.Version = 1
	m.secrets[id] = secret
	m.changelog = append(m.changelog, models.Changelog{
		ID:       m.nextLogID,
		Action:   models.ActionCreate,
		SecretID: id,
		Secret:   secret,
	})
	m.nextLogID++
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
	secret.Version++
	m.secrets[secret.ID] = secret
	m.changelog = append(m.changelog, models.Changelog{
		ID:       m.nextLogID,
		Action:   models.ActionUpdate,
		SecretID: secret.ID,
		Secret:   secret,
	})
	m.nextLogID++
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
	secret, ok := m.secrets[id]
	if !ok {
		return errors.New("данные отсутствуют в хранилище")
	}
	delete(m.secrets, id)
	secret.Version++
	secret.Deleted = true
	m.changelog = append(m.changelog, models.Changelog{
		ID:       m.nextLogID,
		Action:   models.ActionDelete,
		SecretID: secret.ID,
		Secret:   secret,
	})
	m.nextLogID++
	return nil
}

func (m *MemStorage) GetVersion(ctx context.Context) (map[int]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.secrets == nil {
		return nil, errors.New("хранилище пустое")
	}
	versions := make(map[int]int)
	for id, secret := range m.secrets {
		versions[id] = secret.Version
	}
	return versions, nil
}

func (m *MemStorage) GetChangelog() ([]models.Changelog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.changelog == nil {
		return make([]models.Changelog, 0), nil
	}
	changelogCopy := make([]models.Changelog, len(m.changelog))
	copy(changelogCopy, m.changelog)
	return changelogCopy, nil
}

func (m *MemStorage) DeleteChangelog(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var rezult []models.Changelog
	for _, chlog := range m.changelog {
		if chlog.ID > id {
			rezult = append(rezult, chlog)
		}
	}
	m.changelog = rezult
	return nil
}

func (m *MemStorage) SyncUpdate(ctx context.Context, cipherData []models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		m.secrets = make(map[int]models.CipherData)
	}
	for _, cipher := range cipherData {
		if cipher.Deleted {
			delete(m.secrets, cipher.ID)
			continue
		}
		m.secrets[cipher.ID] = cipher
		if cipher.ID >= m.nextID {
			m.nextID = cipher.ID + 1
		}
	}
	return nil
}

// Ping проверяет доступность хранилища.
func (m *MemStorage) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}
