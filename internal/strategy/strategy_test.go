package strategy

import (
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
)

const secretID = "test_seccret"

func TestClientStrategy_Rules(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	tests := []struct {
		name           string
		id             string
		version        int
		updatedAt      time.Time
		secretVersions models.SecretVersions
		wantAction     models.SyncAction
	}{
		{
			name:           "cекрет отсутствует у клиента",
			id:             secretID,
			version:        1,
			updatedAt:      now,
			secretVersions: models.SecretVersions{},
			wantAction:     models.ActionUpdateSecret,
		},
		{
			name:      "версия сервера новее клиентской",
			id:        secretID,
			version:   5,
			updatedAt: now,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 3, UpdatedAt: now},
			},
			wantAction: models.ActionUpdateSecret,
		},
		{
			name:      "версии равны, но время отличается",
			id:        secretID,
			version:   2,
			updatedAt: future,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionConflict,
		},
		{
			name:      "совпадение версии и времени",
			id:        secretID,
			version:   2,
			updatedAt: now,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionSkip,
		},
		{
			name:      "версия сервера старее клиентской",
			id:        secretID,
			version:   1,
			updatedAt: past,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionSkip,
		},
	}
	s := NewClientStrategy()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := s.Rules(tt.id, tt.version, tt.updatedAt, tt.secretVersions)
			assert.Equal(t, tt.wantAction, action)
		})
	}
}

func TestServerStrategy_Rules(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	tests := []struct {
		name           string
		id             string
		version        int
		updatedAt      time.Time
		secretVersions models.SecretVersions
		wantAction     models.SyncAction
	}{
		{
			name:           "секрет отсутствует на сервере",
			id:             secretID,
			version:        1,
			updatedAt:      now,
			secretVersions: models.SecretVersions{},
			wantAction:     models.ActionUpdateSecret,
		},
		{
			name:      "версия клиента новее серверной",
			id:        secretID,
			version:   3,
			updatedAt: now,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 1, UpdatedAt: now},
			},
			wantAction: models.ActionUpdateSecret,
		},
		{
			name:      "версии равны, но время отличается",
			id:        secretID,
			version:   1,
			updatedAt: future,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 1, UpdatedAt: now},
			},
			wantAction: models.ActionConflict,
		},
		{
			name:      "совпадение версии и времени",
			id:        secretID,
			version:   1,
			updatedAt: now,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 1, UpdatedAt: now},
			},
			wantAction: models.ActionSkip,
		},
		{
			name:      "версия клиента старее серверной",
			id:        secretID,
			version:   1,
			updatedAt: past,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionSkip,
		},
	}
	s := NewServerStrategy()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := s.Rules(tt.id, tt.version, tt.updatedAt, tt.secretVersions)
			assert.Equal(t, tt.wantAction, action)
		})
	}
}
