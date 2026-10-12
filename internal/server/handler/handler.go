// Package handler реализует HTTP-интерфейс приложения.
package handler

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/server/sync"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

// Handler связывает HTTP-запросов с хранилищем данных.
type Handler struct {
	storage Storage
	logger  log
	event   sync.Publisher
}

// NewHandlers создает новый экземпляр Handler.
func NewHandlers(storage Storage, l log, event sync.Publisher) *Handler {
	return &Handler{
		storage: storage,
		logger:  l,
		event:   event,
	}
}

// Storage определяет методы для взаимодействия с хранилищем.
type Storage interface {
	SaveUser(ctx context.Context, login, passwordHash, salt string) (int, error)
	GetUser(ctx context.Context, login string) (models.User, error)
	Delete(ctx context.Context, userID, id int) error
	GetSecrets(ctx context.Context, userID int, version models.SecretVersions) ([]models.CipherData, error)
	UpdateSecrets(ctx context.Context, userID int, changelogs models.Changelogs) error
}
