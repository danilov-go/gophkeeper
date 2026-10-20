package memory

import (
	"context"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (m *MemStorage) SaveUser(ctx context.Context, login, hash, salt string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	user := models.LocalUser{
		Hash: hash,
		Salt: salt,
	}
	m.user[login] = user
	return nil
}

func (m *MemStorage) GetUser(ctx context.Context, login string) (models.LocalUser, error) {
	var user models.LocalUser
	if err := ctx.Err(); err != nil {
		return user, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.user == nil {
		return user, errors.New("хранилище пустое")
	}
	salt, ok := m.user[login]
	if !ok {
		return user, errors.New("пользователь отсутствует локально")
	}
	return salt, nil
}
