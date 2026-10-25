package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const (
	secretID1 = "secretID1"
	secretID2 = "secretID2"
)

func TestMemStorage_GetVersion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	tests := []struct {
		name          string
		setupSecrets  []models.CipherData
		cancelContext bool
		want          map[string]models.VersionTime
		wantErr       bool
		errText       string
	}{
		{
			name: "положительный тест",
			setupSecrets: []models.CipherData{
				{ID: secretID1, Version: 3, UpdatedAt: now, Type: models.TypeCard},
				{ID: secretID2, Version: 1, UpdatedAt: now, Type: models.TypeBinary},
			},
			want: map[string]models.VersionTime{
				secretID1: {Version: 3, UpdatedAt: now},
				secretID2: {Version: 1, UpdatedAt: now},
			},
			wantErr: false,
		},
		{
			name:          "ошибка отмены контекста",
			setupSecrets:  nil,
			cancelContext: true,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()
			for _, s := range tt.setupSecrets {
				_, err := storage.SaveSecret(ctx, s)
				assert.NoError(t, err)
				secret, err := storage.GetSecret(ctx, s.ID)
				assert.NoError(t, err)
				secret.Version = s.Version
				secret.UpdatedAt = s.UpdatedAt
				err = storage.SyncUpdate(ctx, []models.CipherData{secret})
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			versions, err := storage.GetVersion(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, versions)
		})
	}
}

func TestMemStorage_Changelog(t *testing.T) {
	storage := memory.InitMemStorage()
	ctx := t.Context()
	changelogs, err := storage.GetChangelog()
	assert.NoError(t, err)
	assert.Len(t, changelogs, 0)
	id1, err := storage.SaveSecret(ctx, models.CipherData{Type: models.TypeCard})
	assert.NoError(t, err)
	id2, err := storage.SaveSecret(ctx, models.CipherData{Type: models.TypeBinary})
	assert.NoError(t, err)
	changelog, err := storage.GetChangelog()
	assert.NoError(t, err)
	assert.Len(t, changelog, 2)
	assert.Equal(t, id1, changelog[0].SecretID)
	assert.Equal(t, id2, changelog[1].SecretID)
	id := changelog[0].ID
	err = storage.DeleteChangelog(id)
	assert.NoError(t, err)
	remainingLogs, err := storage.GetChangelog()
	assert.NoError(t, err)
	assert.Len(t, remainingLogs, 1)
	assert.Equal(t, id2, remainingLogs[0].SecretID)

}

func TestMemStorage_SyncUpdate(t *testing.T) {
	tests := []struct {
		name          string
		secret        models.CipherData
		secrets       []models.CipherData
		cancelContext bool
		wantExist     bool
		wantCipher    []byte
	}{
		{
			name: "положительный тест",
			secrets: []models.CipherData{
				{ID: secretID1, Type: models.TypeCard, Cipher: []byte("test")},
			},
			wantExist:  true,
			wantCipher: []byte("test"),
		},
		{
			name:   "удаление секрета",
			secret: models.CipherData{ID: secretID1, Type: models.TypeBinary, Cipher: []byte("test")},
			secrets: []models.CipherData{
				{ID: secretID1, Deleted: true},
			},
			wantExist: false,
		},
		{
			name: "ошибка отмены контекста",
			secrets: []models.CipherData{
				{ID: secretID1, Type: models.TypeCard},
			},
			cancelContext: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := memory.InitMemStorage()
			ctx := t.Context()
			if tt.secret.ID != "" {
				_, err := storage.SaveSecret(ctx, tt.secret)
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := storage.SyncUpdate(ctx, tt.secrets)
			if tt.cancelContext {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			for _, expSecret := range tt.secrets {
				secret, err := storage.GetSecret(t.Context(), expSecret.ID)
				if tt.wantExist {
					assert.NoError(t, err)
					assert.Equal(t, tt.wantCipher, secret.Cipher)
				} else {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "данные отсутствуют в хранилище")
				}
			}
		})
	}
}
