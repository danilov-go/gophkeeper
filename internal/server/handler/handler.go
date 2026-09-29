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
	Ping(ctx context.Context) error
	SaveUser(ctx context.Context, login, passwordHash string) (int, error)
	GetUser(ctx context.Context, login string) (models.User, error)
	Save(ctx context.Context, login string, cipherData models.CipherData) (int, error)
	Get(ctx context.Context, userID, id int) (models.CipherData, error)
	GetAll(ctx context.Context, userID int) ([]models.CipherData, error)
	Delete(ctx context.Context, userID, id int) error
}
