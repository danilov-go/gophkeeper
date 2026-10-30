package service_test

import (
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/server/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/server/service"
	"github.com/danilov-go/gophkeeper/internal/server/sync"
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
	jwtKey    = "super-secret-key-12345"
)

func TestServerService_Register(t *testing.T) {
	tests := []struct {
		name   string
		login  string
		hash   string
		salt   string
		expErr error
	}{
		{
			name:   "положительный тест",
			login:  expLogin,
			hash:   expHash,
			salt:   expSalt,
			expErr: nil,
		},
		{
			name:   "пользователь уже существует",
			login:  expLogin,
			hash:   expHash,
			salt:   expSalt,
			expErr: models.ErrUserAlreadyExists,
		},
	}
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
			ctx := t.Context()
			token, err := svc.Register(ctx, tt.login, tt.hash, tt.salt)
			if tt.expErr != nil {
				assert.ErrorIs(t, err, tt.expErr)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestServerService_Login(t *testing.T) {
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
	ctx := t.Context()
	_, err := storage.SaveUser(ctx, expLogin, expHash, expSalt)
	assert.NoError(t, err)
	tests := []struct {
		name       string
		login      string
		hash       string
		expErr     bool
		expectSalt string
	}{
		{
			name:       "положительный тест",
			login:      expLogin,
			hash:       expHash,
			expErr:     false,
			expectSalt: expSalt,
		},
		{
			name:   "пользователь не найден",
			login:  "unknow",
			hash:   expHash,
			expErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, salt, err := svc.Login(ctx, tt.login, tt.hash)
			if tt.expErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				if err == nil {
					assert.NoError(t, err)
					assert.NotEmpty(t, token)
					assert.Equal(t, tt.expectSalt, salt)
				}
			}
		})
	}
}

func TestServerService_GetSalt(t *testing.T) {
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
	ctx := t.Context()
	_, err := storage.SaveUser(ctx, expLogin, expHash, expSalt)
	assert.NoError(t, err)
	tests := []struct {
		name          string
		login         string
		expectSalt    string
		expectSaltLen int
		isRealUser    bool
	}{
		{
			name:          "положительный тест",
			login:         expLogin,
			expectSalt:    expSalt,
			expectSaltLen: 9,
			isRealUser:    true,
		},
		{
			name:          "пользователь не найден",
			login:         "unknown",
			expectSaltLen: 32,
			isRealUser:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			salt := svc.GetSalt(ctx, tt.login)
			assert.Len(t, salt, tt.expectSaltLen)
			if tt.isRealUser {
				assert.Equal(t, tt.expectSalt, salt)
			}
		})
	}
}

func TestServerService_GetUserLogin(t *testing.T) {
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
	validToken, err := svc.BuildJWTString(userID, expLogin)
	assert.NoError(t, err)
	tests := []struct {
		name        string
		tokenString string
		expErr      bool
		expID       int
		expLogin    string
	}{
		{
			name:        "положительный тест",
			tokenString: validToken,
			expErr:      false,
			expID:       userID,
			expLogin:    expLogin,
		},
		{
			name:        "невалидная строка токена",
			tokenString: "invalide_token",
			expErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authUser, err := svc.GetUserLogin(tt.tokenString)
			if tt.expErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expID, authUser.ID)
				assert.Equal(t, tt.expLogin, authUser.Login)
			}
		})
	}
}

func TestServerService_Push(t *testing.T) {
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
	ctx := t.Context()
	now := time.Now().Round(0)
	tests := []struct {
		name       string
		deviceID   string
		changelogs models.Changelogs
	}{
		{
			name:     "положительный тест",
			deviceID: "deviceID",
			changelogs: models.Changelogs{
				{
					SecretID: secretID1,
					Secret:   models.CipherData{ID: secretID1, Version: 1, UpdatedAt: now, Cipher: []byte("secret")},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventChan, closeChan := svc.Event(ctx, userID, "device")
			defer closeChan()
			time.Sleep(10 * time.Millisecond)
			err := svc.Push(ctx, userID, tt.deviceID, tt.changelogs)
			assert.NoError(t, err)
			select {
			case event := <-eventChan:
				assert.Equal(t, userID, event.UserID)
				assert.Equal(t, tt.deviceID, event.DeviceID)
				assert.Equal(t, "sync", event.Message)
			case <-time.After(500 * time.Millisecond):
				t.Fatal("уведомления не дошло до канала")
			}
		})
	}
}

func TestServerService_Pull(t *testing.T) {
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
	ctx := t.Context()
	now := time.Now()
	err := storage.UpdateSecrets(ctx, userID, models.Changelogs{
		{
			SecretID: secretID1,
			Secret:   models.CipherData{ID: secretID1, Version: 2, UpdatedAt: now, Cipher: []byte("secret")},
		},
	})
	assert.NoError(t, err)
	tests := []struct {
		name           string
		userID         int
		clientVersions models.SecretVersions
		count          int
	}{
		{
			name:   "версия клиента выше версии сервера",
			userID: userID,
			clientVersions: models.SecretVersions{
				"vault_item": models.VersionTime{Version: 1, UpdatedAt: now},
			},
			count: 1,
		},
		{
			name:   "версии сервера и клиента совпадают",
			userID: userID,
			clientVersions: models.SecretVersions{
				secretID1: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			count: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.Pull(ctx, tt.userID, tt.clientVersions)
			assert.NoError(t, err)
			assert.Len(t, res, tt.count)
		})
	}
}

func TestServerService_Event(t *testing.T) {
	storage := memory.InitMemStorage(strategy.NewServerStrategy())
	svc := service.NewServerService(storage, jwtKey, sync.NewSyncNotifier())
	ctx := t.Context()
	ch, closeFunc := svc.Event(ctx, 15, "deviceID")
	assert.NotNil(t, ch)
	assert.NotNil(t, closeFunc)
	assert.NotPanics(t, func() {
		closeFunc()
	})
}
