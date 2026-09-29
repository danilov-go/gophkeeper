package client

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

type Client interface {
	Register(ctx context.Context, login, password string) error
	Auth(ctx context.Context, login, password string) error
	SaveSecret(ctx context.Context, secret models.SecretData) error
	GetSecret(ctx context.Context, id int) (models.SecretData, error)
	GetAllSecrets(ctx context.Context) ([]models.CipherData, error)
}

type Storage interface {
	Save(ctx context.Context, cipherData models.CipherData) (int, error)
	UpdateID(ctx context.Context, localID, ID int) error
	SaveAll(ctx context.Context, cipherData []models.CipherData) error
}

func New(cfg config.ConfigClient, l log, storage Storage, cryptoKey []byte) Client {
	return NewHTTPClient(cfg, l, storage, cryptoKey)
}
