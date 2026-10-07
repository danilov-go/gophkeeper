package memory

import (
	"context"
	"sync"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type MemStorage struct {
	mu         sync.RWMutex
	nextID     int
	usersID    map[int]models.User
	usersLogin map[string]int
	secrets    map[int]map[int]models.CipherData
}

func InitMemStorage() *MemStorage {
	return &MemStorage{
		nextID:     1,
		usersID:    make(map[int]models.User),
		usersLogin: make(map[string]int),
		secrets:    make(map[int]map[int]models.CipherData),
	}
}

func (m *MemStorage) GetSecrets(ctx context.Context, userID int, v models.SecretVersions) ([]models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	secrets, ok := m.secrets[userID]
	if !ok || len(secrets) == 0 {
		return nil, nil
	}
	var cipherData []models.CipherData
	for id, secret := range secrets {
		version, ok := v[id]
		if !ok || secret.Version > version {
			cipherData = append(cipherData, secret)
		}
	}
	return cipherData, nil
}

func (m *MemStorage) UpdateSecrets(ctx context.Context, userID int, changelogs models.Changelogs) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.secrets == nil {
		m.secrets = make(map[int]map[int]models.CipherData)
	}
	if _, ok := m.secrets[userID]; !ok {
		m.secrets[userID] = make(map[int]models.CipherData)
	}
	for _, changelog := range changelogs {
		secret := changelog.Secret
		id := changelog.SecretID
		serverSecret, exists := m.secrets[userID][id]
		if !exists {
			secret.UpdatedAt = time.Now()
			m.secrets[userID][id] = secret
		} else {
			if secret.Version > serverSecret.Version {
				secret.UpdatedAt = time.Now()
				m.secrets[userID][id] = secret
			}
		}
	}
	return nil
}

// SaveUser сохраняет нового пользователя.
func (m *MemStorage) SaveUser(ctx context.Context, login, passwordHash string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.usersLogin[login]; ok {
		return 0, models.ErrUserAlreadyExists
	}
	userID := m.nextID
	m.nextID++
	newUser := models.User{
		ID:           userID,
		Login:        login,
		PasswordHash: passwordHash,
	}
	m.usersID[userID] = newUser
	m.usersLogin[login] = userID
	return userID, nil
}

// GetUser возвращает данные пользователя.
func (m *MemStorage) GetUser(ctx context.Context, login string) (models.User, error) {
	if err := ctx.Err(); err != nil {
		return models.User{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	userID, ok := m.usersLogin[login]
	if !ok {
		return models.User{}, models.ErrUserNotFound
	}
	user, ok := m.usersID[userID]
	if !ok {
		return models.User{}, models.ErrUserNotFound
	}
	return user, nil
}

func (m *MemStorage) Delete(ctx context.Context, userID, id int) error {
	return nil
}
