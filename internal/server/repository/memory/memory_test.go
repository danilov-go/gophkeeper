package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/server/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/strategy"

	"github.com/stretchr/testify/assert"
)

const (
	expLogin  = "login"
	expHash   = "password"
	expSalt   = "test_salt"
	userID    = 15
	secretID1 = "test1"
	secretID2 = "test2"
)

func TestMemStorage_SaveUser(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		hash    string
		salt    string
		id      int
		wantErr bool
	}{
		{
			name:    "положительный тест",
			login:   expLogin,
			hash:    expHash,
			salt:    expSalt,
			id:      1,
			wantErr: false,
		},
		{
			name:    "логин занят",
			login:   expLogin,
			hash:    expHash,
			salt:    expSalt,
			id:      0,
			wantErr: true,
		},
	}
	strategy := strategy.NewServerStrategy()
	storage := memory.InitMemStorage(strategy)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := storage.SaveUser(t.Context(), tt.login, tt.hash, tt.salt)
			assert.Equal(t, tt.id, id)
			if tt.wantErr {
				assert.ErrorIs(t, err, models.ErrUserAlreadyExists)
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func TestMemStorage_GetUser(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		want    models.User
		wantErr bool
	}{
		{
			name:  "положительный тест",
			login: expLogin,
			want: models.User{
				ID:    1,
				Login: expLogin,
				Hash:  expHash,
				Salt:  expSalt,
			},
			wantErr: false,
		},
		{
			name:    "пользователь не найден",
			login:   "unknown",
			want:    models.User{},
			wantErr: true,
		},
	}
	strategy := strategy.NewServerStrategy()
	storage := memory.InitMemStorage(strategy)
	_, err := storage.SaveUser(t.Context(), expLogin, expHash, expSalt)
	assert.NoError(t, err)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := storage.GetUser(t.Context(), tt.login)
			if tt.wantErr {
				assert.ErrorIs(t, err, models.ErrUserNotFound)
				assert.Equal(t, tt.want, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, user)
			}
		})
	}
}

func TestMemStorage_GetSecrets(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	secret1 := models.CipherData{ID: secretID1, Version: 1, UpdatedAt: now}
	secret2 := models.CipherData{ID: secretID2, Version: 2, UpdatedAt: now}
	tests := []struct {
		name           string
		clientVersions models.SecretVersions
		wantLen        int
		checkID        string
	}{
		{
			name: "у клиента актуальные версии",
			clientVersions: models.SecretVersions{
				secretID1: models.VersionTime{Version: 1, UpdatedAt: now},
				secretID2: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantLen: 0,
		},
		{
			name: "у клиента старая версия или нет данных",
			clientVersions: models.SecretVersions{
				secretID2: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantLen: 1,
			checkID: secretID1,
		},
		{
			name:           "у клиента нет данных",
			clientVersions: models.SecretVersions{},
			wantLen:        2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			realStrategy := strategy.NewServerStrategy()
			storage := memory.InitMemStorage(realStrategy)
			ctx := t.Context()
			changelogs := models.Changelogs{
				{SecretID: secretID1, Secret: secret1},
				{SecretID: secretID2, Secret: secret2},
			}
			err := storage.UpdateSecrets(ctx, 1, changelogs)
			assert.NoError(t, err)
			res, err := storage.GetSecrets(ctx, 1, tt.clientVersions)
			assert.NoError(t, err)
			if !assert.Len(t, res, tt.wantLen) {
				return
			}
			if tt.wantLen == 1 && tt.checkID != "" {
				assert.Equal(t, tt.checkID, res[0].ID)
			}
		})
	}
}

func TestMemStorage_UpdateSecrets(t *testing.T) {
	now := time.Now()
	secret1 := models.CipherData{ID: secretID1, Version: 1, UpdatedAt: now}
	secret2 := models.CipherData{ID: secretID1, Version: 2, UpdatedAt: now}
	tests := []struct {
		name          string
		configStorage models.Changelogs
		updateLogs    models.Changelogs
		cancelContext bool
		wantVersion   int
		wantErr       bool
	}{
		{
			name:          "положительный тест",
			configStorage: nil,
			updateLogs: models.Changelogs{
				{SecretID: secretID1, Secret: secret1},
			},
			wantVersion: 1,
			wantErr:     false,
		},
		{
			name: "обновление на более новую версию",
			configStorage: models.Changelogs{
				{SecretID: secretID1, Secret: secret1},
			},
			updateLogs: models.Changelogs{
				{SecretID: secretID1, Secret: secret2},
			},
			wantVersion: 2,
			wantErr:     false,
		},
		{
			name: "пропуск секрета",
			configStorage: models.Changelogs{
				{SecretID: secretID1, Secret: secret2},
			},
			updateLogs: models.Changelogs{
				{SecretID: secretID1, Secret: secret1},
			},
			wantVersion: 2,
			wantErr:     false,
		},
		{
			name:          "ошибка отмены контекста",
			configStorage: nil,
			updateLogs:    models.Changelogs{{SecretID: secretID1, Secret: secret1}},
			cancelContext: true,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := strategy.NewServerStrategy()
			storage := memory.InitMemStorage(strategy)
			ctx := t.Context()
			if tt.configStorage != nil {
				err := storage.UpdateSecrets(ctx, userID, tt.configStorage)
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := storage.UpdateSecrets(ctx, userID, tt.updateLogs)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			secrets, err := storage.GetSecrets(t.Context(), userID, models.SecretVersions{})
			assert.NoError(t, err)
			if assert.Len(t, secrets, 1) {
				assert.Equal(t, tt.wantVersion, secrets[0].Version)
			}
		})
	}
}
