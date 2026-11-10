package memory

import (
	"context"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// GetVersion возвращает версии и время обновления всех секретов.
func (m *MemStorage) GetVersion(ctx context.Context) (map[string]models.VersionTime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.secrets == nil {
		return nil, errors.New("хранилище пустое")
	}
	versions := make(map[string]models.VersionTime)
	for id, secret := range m.secrets {
		versions[id] = models.VersionTime{
			Version:   secret.Version,
			UpdatedAt: secret.UpdatedAt,
		}
	}
	return versions, nil
}

// GetChangelog возвращает все записи из журнала изменений.
func (m *MemStorage) GetChangelog(ctx context.Context) ([]models.Changelog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.changelog == nil {
		return make([]models.Changelog, 0), nil
	}
	changelogCopy := make([]models.Changelog, len(m.changelog))
	copy(changelogCopy, m.changelog)
	return changelogCopy, nil
}

// DeleteChangelog удаляет обработанные записи журнала изменений до указанного ID.
func (m *MemStorage) DeleteChangelog(ctx context.Context, id int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	i := 0
	for _, log := range m.changelog {
		if log.ID > id {
			m.changelog[i] = log
			i++
		}
	}
	for j := i; j < len(m.changelog); j++ {
		m.changelog[j] = models.Changelog{}
	}
	m.changelog = m.changelog[:i]
	return nil
}

// SyncUpdate сохраняет или обновляет полученные с сервера секреты.
func (m *MemStorage) SyncUpdate(ctx context.Context, cipherData []models.CipherData) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secrets == nil {
		m.secrets = make(map[string]models.CipherData)
	}
	for _, cipher := range cipherData {
		if cipher.Deleted {
			delete(m.secrets, cipher.ID)
			continue
		}
		m.secrets[cipher.ID] = cipher

	}
	return nil
}
