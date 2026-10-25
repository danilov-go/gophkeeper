package memory_test

import (
	"context"
	"testing"

	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const (
	secretID = "secretID"
)

func TestMemStorage_SaveSecret(t *testing.T) {
	tests := []struct {
		name          string
		secret        models.CipherData
		cancelContext bool
		wantErr       bool
	}{
		{
			name: "положительный тест",
			secret: models.CipherData{
				ID:     secretID,
				Type:   models.TypeLoginPassword,
				Cipher: []byte("test"),
			},
			wantErr: false,
		},
		{
			name: "успешное сохранение с автогенерацией ID",
			secret: models.CipherData{
				ID:     "",
				Type:   models.TypeCard,
				Cipher: []byte("test"),
			},
			wantErr: false,
		},
		{
			name: "ошибка отмены контекста",
			secret: models.CipherData{
				ID: secretID,
			},
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
			id, err := storage.SaveSecret(ctx, tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, id)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, id)

			if tt.secret.ID != "" {
				assert.Equal(t, tt.secret.ID, id)
			}
			secret, err := storage.GetSecret(t.Context(), id)
			assert.NoError(t, err)
			assert.Equal(t, 1, secret.Version)
			assert.Equal(t, tt.secret.Type, secret.Type)
			assert.Equal(t, tt.secret.Cipher, secret.Cipher)
		})
	}
}

func TestMemStorage_GetSecret(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		setupSecret   bool
		cancelContext bool
		wantErr       bool
		errText       string
	}{
		{
			name:        "положительный тест",
			id:          secretID,
			setupSecret: true,
			wantErr:     false,
		},
		{
			name:        "данные отсутствуют",
			id:          "unknow",
			setupSecret: true,
			wantErr:     true,
			errText:     "данные отсутствуют в хранилище",
		},
		{
			name:          "ошибка отмены контекста",
			id:            secretID,
			setupSecret:   true,
			cancelContext: true,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()
			if tt.setupSecret {
				_, err := storage.SaveSecret(ctx, models.CipherData{ID: secretID, Type: models.TypeBinary})
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			secret, err := storage.GetSecret(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errText != "" {
					assert.Contains(t, err.Error(), tt.errText)
				}
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.id, secret.ID)
		})
	}
}

func TestMemStorage_GetAllSecret(t *testing.T) {
	tests := []struct {
		name       string
		searchType models.SecretType
		setupData  []models.CipherData
		wantCount  int
		wantErr    bool
		errText    string
	}{
		{
			name:       "положительный тест",
			searchType: models.TypeBinary,
			setupData: []models.CipherData{
				{ID: "1", Type: models.TypeCard},
				{ID: "2", Type: models.TypeBinary},
				{ID: "3", Type: models.TypeLoginPassword},
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:       "отсутствуют данные такого типа",
			searchType: models.TypeText,
			setupData: []models.CipherData{
				{ID: "1", Type: models.TypeCard},
			},
			wantErr: true,
			errText: "данные отсутствуют в хранилище",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()
			for _, s := range tt.setupData {
				_, err := storage.SaveSecret(ctx, s)
				assert.NoError(t, err)
			}
			secrets, err := storage.GetAllSecret(ctx, tt.searchType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, secrets, tt.wantCount)
			for _, s := range secrets {
				assert.Equal(t, tt.searchType, s.Type)
			}
		})
	}
}

func TestMemStorage_UpdateSecret(t *testing.T) {
	tests := []struct {
		name          string
		setupSecret   bool
		secretToUpd   models.CipherData
		cancelContext bool
		wantErr       bool
		errText       string
	}{
		{
			name:        "положительный тест",
			setupSecret: true,
			secretToUpd: models.CipherData{
				ID:     secretID,
				Type:   models.TypeCard,
				Cipher: []byte("test"),
			},
			wantErr: false,
		},
		{
			name:        "обновление несуществующего секрета",
			setupSecret: false,
			secretToUpd: models.CipherData{
				ID: "unknown",
			},
			wantErr: true,
			errText: "данные отсутствуют в хранилище",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()
			if tt.setupSecret {
				_, err := storage.SaveSecret(ctx, models.CipherData{ID: secretID, Type: models.TypeCard, Cipher: []byte("secret")})
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := storage.UpdateSecret(ctx, tt.secretToUpd)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			assert.NoError(t, err)
			updated, err := storage.GetSecret(t.Context(), secretID)
			assert.NoError(t, err)
			assert.Equal(t, 2, updated.Version)
			assert.Equal(t, tt.secretToUpd.Cipher, updated.Cipher)
		})
	}
}

func TestMemStorage_DeleteSecret(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		setupSync   bool
		setupSecret bool
		wantErr     bool
		errText     string
	}{
		{
			name:        "положительный тест (sync = false)",
			id:          secretID,
			setupSync:   false,
			setupSecret: true,
			wantErr:     false,
		},
		{
			name:        "положительный тест (sync = true)",
			id:          secretID,
			setupSync:   true,
			setupSecret: true,
			wantErr:     false,
		},
		{
			name:        "удаление отсутствующего секрета",
			id:          "unknown",
			setupSync:   false,
			setupSecret: false,
			wantErr:     true,
			errText:     "данные отсутствуют в хранилище",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()

			if tt.setupSecret {
				_, err := storage.SaveSecret(ctx, models.CipherData{ID: secretID, Type: models.TypeCard})
				assert.NoError(t, err)
			}
			err := storage.DeleteSecret(ctx, tt.id, tt.setupSync)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errText)
				return
			}
			assert.NoError(t, err)
			_, err = storage.GetSecret(t.Context(), tt.id)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "данные отсутствуют в хранилище")
		})
	}
}
