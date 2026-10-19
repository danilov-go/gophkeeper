package memory

import (
	"context"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (m *MemStorage) GetVersion(ctx context.Context) (map[int]models.VersionTime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.secrets == nil {
		return nil, errors.New("хранилище пустое")
	}
	versions := make(map[int]models.VersionTime)
	for id, secret := range m.secrets {
		versions[id] = models.VersionTime{
			Version:   secret.Version,
			UpdatedAt: secret.UpdatedAt,
		}
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
