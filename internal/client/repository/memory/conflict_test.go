package memory_test

import (
	"context"
	"testing"

	"github.com/danilov-go/gophkeeper/internal/client/repository/memory"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestMemStorage_SaveConflict(t *testing.T) {
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
				Cipher: []byte("conflict"),
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
			err := storage.SaveConflict(ctx, tt.secret)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if !tt.cancelContext {
				conflict, err := storage.GetConflicts(t.Context())
				assert.NoError(t, err)
				if assert.Len(t, conflict, 1) {
					assert.Equal(t, tt.secret.ID, conflict[0].ID)
					assert.Equal(t, tt.secret.Cipher, conflict[0].Cipher)
				}
			}
		})
	}
}

func TestMemStorage_GetConflicts(t *testing.T) {
	tests := []struct {
		name          string
		setupSecrets  []models.CipherData
		cancelContext bool
		wantCount     int
		wantErr       bool
	}{
		{
			name: "положительный тест",
			setupSecrets: []models.CipherData{
				{ID: secretID1, Type: models.TypeCard},
				{ID: secretID2, Type: models.TypeBinary},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:         "получение конфликтов из пустого хранилища",
			setupSecrets: nil,
			wantCount:    0,
			wantErr:      false,
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
				err := storage.SaveConflict(ctx, s)
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			conflicts, err := storage.GetConflicts(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, conflicts, tt.wantCount)
			if tt.wantCount > 0 {
				var conflictID []string
				for _, conflict := range conflicts {
					conflictID = append(conflictID, conflict.ID)
				}
				for _, secret := range tt.setupSecrets {
					assert.Contains(t, conflictID, secret.ID)
				}
			}
		})
	}
}

func TestMemStorage_DeleteConflict(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		setupSecret   bool
		cancelContext bool
		wantErr       bool
	}{
		{
			name:        "успешное удаление существующего конфликта",
			id:          secretID,
			setupSecret: true,
			wantErr:     false,
		},
		{
			name:        "удаление несуществующего конфликта",
			id:          "non-existent-id",
			setupSecret: true,
			wantErr:     false,
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
				err := storage.SaveConflict(ctx, models.CipherData{ID: secretID, Type: models.TypeCard})
				assert.NoError(t, err)
			}
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := storage.DeleteConflict(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if !tt.cancelContext {
				conflicts, err := storage.GetConflicts(t.Context())
				assert.NoError(t, err)
				if tt.id == secretID {
					assert.Len(t, conflicts, 0)
				} else {
					assert.Len(t, conflicts, 1)
					assert.Equal(t, secretID, conflicts[0].ID)
				}
			}
		})
	}
}
