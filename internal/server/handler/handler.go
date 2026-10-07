// Package handler реализует HTTP-интерфейс приложения.
package handler

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

// Handler связывает HTTP-запросов с хранилищем данных.
type Handler struct {
	storage Storage
	logger  log
}

// NewHandlers создает новый экземпляр Handler.
func NewHandlers(storage Storage, l log) *Handler {
	return &Handler{
		storage: storage,
		logger:  l,
	}
}

// Storage определяет методы для взаимодействия с хранилищем.
type Storage interface {
	SaveUser(ctx context.Context, login, passwordHash string) (int, error)
	GetUser(ctx context.Context, login string) (models.User, error)
	Delete(ctx context.Context, userID, id int) error
	GetSecrets(ctx context.Context, userID int, version models.SecretVersions) ([]models.CipherData, error)
	UpdateSecrets(ctx context.Context, userID int, changelogs models.Changelogs) error
}
