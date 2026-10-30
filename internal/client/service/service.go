package service

import (
	"context"
	"sync"
	"time"

	"github.com/awnumar/memguard"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// Storage определяет методы для взаимодействия с хранилищем.
//
//go:generate mockery --name=Storage --case=underscore --output=./mocks
type Storage interface {
	SaveSecret(ctx context.Context, secret models.CipherData) (string, error)
	GetSecret(ctx context.Context, id string) (models.CipherData, error)
	GetAllSecret(ctx context.Context, sType models.SecretType) ([]models.CipherData, error)
	UpdateSecret(ctx context.Context, secret models.CipherData) error
	DeleteSecret(ctx context.Context, id string, sync bool) error
	SaveUser(ctx context.Context, login, hash, salt string) error
	GetUser(ctx context.Context, login string) (models.LocalUser, error)
	SyncUpdate(ctx context.Context, cipherData []models.CipherData) error
	GetVersion(ctx context.Context) (map[string]models.VersionTime, error)
	GetChangelog() ([]models.Changelog, error)
	DeleteChangelog(id int) error
	SaveConflict(ctx context.Context, secret models.CipherData) error
	GetConflicts(ctx context.Context) ([]models.CipherData, error)
	DeleteConflict(ctx context.Context, id string) error
}

//go:generate mockery --name=Sender --case=underscore --output=./mocks
type Strategy interface {
	Rules(id string, version int, updatedAt time.Time, deleted bool, versions models.SecretVersions) models.SyncAction
}

//go:generate mockery --name=Strategy --case=underscore --output=./mocks
type Sender interface {
	Register(ctx context.Context, login, hash, salt string) error
	Auth(ctx context.Context, login, hash string) error
	Sync(ctx context.Context, versions map[string]models.VersionTime, changelogs []models.Changelog) ([]models.CipherData, error)
	Logout()
	ServerEvent(ctx context.Context, runSync func())
	GetSalt(ctx context.Context, login string) (string, error)
}

// ClientService управляет бизнес-логикой.
type ClientService struct {
	mu       sync.RWMutex
	storage  Storage
	sender   Sender
	strategy Strategy
	key      *memguard.LockedBuffer
	syncCh   chan<- struct{}
}

// NewClientService создает экземпляр ClientService.
func NewClientService(storage Storage, sender Sender, strategy Strategy, syncCh chan<- struct{}) *ClientService {
	return &ClientService{
		storage:  storage,
		sender:   sender,
		strategy: strategy,
		syncCh:   syncCh,
	}
}
