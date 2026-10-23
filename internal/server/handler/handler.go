// Package handler реализует HTTP-интерфейс приложения.
package handler

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

//go:generate mockery --name=ServerService --output=./mocks --case=underscore
type ServerService interface {
	BuildJWTString(id int, login string) (string, error)
	Register(ctx context.Context, login, clientHash, clientSalt string) (string, error)
	Login(ctx context.Context, login, clientHash string) (string, string, error)
	GetSalt(ctx context.Context, login string) string
	GetUserLogin(token string) (models.AuthUser, error)
	Push(ctx context.Context, userID int, deviceID string, changelogs models.Changelogs) error
	Pull(ctx context.Context, userID int, versions models.SecretVersions) ([]models.CipherData, error)
	Event(ctx context.Context, userID int, deviceID string) (<-chan models.SyncEvent, func())
}

// Handler связывает HTTP-запросов с бизнес логикой.
type Handler struct {
	logger  log
	service ServerService
}

// NewHandlers создает новый экземпляр Handler.
func NewHandlers(l log, service ServerService) *Handler {
	return &Handler{
		logger:  l,
		service: service,
	}
}
