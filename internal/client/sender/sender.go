package sender

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

type Sender interface {
	Register(ctx context.Context, login, password string) error
	Auth(ctx context.Context, login, password string) error
	Sync(ctx context.Context) error
}

type Storage interface {
	GetVersion(ctx context.Context) (map[int]int, error)
	SyncUpdate(ctx context.Context, cipherData []models.CipherData) error
	GetChangelog() ([]models.Changelog, error)
	DeleteChangelog(id int) error
}

func NewSender(cfg config.ConfigClient, l log, storage Storage) Sender {
	return NewHTTPSender(cfg, l, storage)
}
