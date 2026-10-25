package memory_test

import (
	"context"
	"testing"

	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const (
	expLogin = "login"
	expHash  = "hash"
	expSalt  = "salt"
)

func TestMemStorage_SaveUser(t *testing.T) {
	tests := []struct {
		name          string
		login         string
		hash          string
		salt          string
		cancelContext bool
		wantErr       bool
	}{
		{
			name:    "положительный тест",
			login:   expLogin,
			hash:    expHash,
			salt:    expSalt,
			wantErr: false,
		},
		{
			name:    "перезапись существующего пользователя",
			login:   expLogin,
			hash:    "new_hash",
			salt:    "new_salt",
			wantErr: false,
		},
		{
			name:          "ошибка отмены контекста",
			login:         expLogin,
			hash:          expHash,
			salt:          expSalt,
			cancelContext: true,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := storage.SaveUser(ctx, tt.login, tt.hash, tt.salt)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if !tt.cancelContext {
				user, err := storage.GetUser(t.Context(), tt.login)
				assert.NoError(t, err)
				assert.Equal(t, tt.hash, user.Hash)
				assert.Equal(t, tt.salt, user.Salt)
			}
		})
	}
}

func TestMemStorage_GetUser(t *testing.T) {
	tests := []struct {
		name          string
		login         string
		setupUser     bool
		cancelContext bool
		want          models.LocalUser
		wantErr       bool
		errText       string
	}{
		{
			name:      "положительный тест",
			login:     expLogin,
			setupUser: true,
			want: models.LocalUser{
				Hash: expHash,
				Salt: expSalt,
			},
			wantErr: false,
		},
		{
			name:      "пользователь отсутствует",
			login:     "unknown_local",
			setupUser: false,
			want:      models.LocalUser{},
			wantErr:   true,
			errText:   "пользователь отсутствует локально",
		},
		{
			name:          "ошибка отмены контекста",
			login:         expLogin,
			setupUser:     true,
			cancelContext: true,
			want:          models.LocalUser{},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()

			if tt.setupUser {
				err := storage.SaveUser(ctx, expLogin, expHash, expSalt)
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			user, err := storage.GetUser(ctx, tt.login)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.Contains(t, err.Error(), tt.errText)
				}
				assert.Equal(t, tt.want, user)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, user)
		})
	}
}
