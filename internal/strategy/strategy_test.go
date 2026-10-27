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
		deleted        bool
		secretVersions models.SecretVersions
		wantAction     models.SyncAction
	}{
		{
			name:           "cекрет отсутствует у клиента",
			id:             secretID,
			version:        1,
			updatedAt:      now,
			deleted:        false,
			secretVersions: models.SecretVersions{},
			wantAction:     models.ActionUpdate,
		},
		{
			name:      "версия сервера новее клиентской",
			id:        secretID,
			version:   5,
			updatedAt: now,
			deleted:   false,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 3, UpdatedAt: now},
			},
			wantAction: models.ActionUpdate,
		},
		{
			name:      "версии равны, но время отличается",
			id:        secretID,
			version:   2,
			updatedAt: future,
			deleted:   false,
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
			deleted:   false,
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
			deleted:   false,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionSkip,
		},
		{
			name:           "секрет отсутствует у клиента и помечен удаленным на сервере",
			id:             secretID,
			version:        1,
			updatedAt:      now,
			deleted:        true,
			secretVersions: models.SecretVersions{},
			wantAction:     models.ActionSkip,
		},
		{
			name:      "версия сервера новее клиентской и секрет удален",
			id:        secretID,
			version:   5,
			updatedAt: now,
			deleted:   true,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 3, UpdatedAt: now},
			},
			wantAction: models.ActionDelete,
		},
		{
			name:      "версии равны, время отличается и секрет удален на сервере",
			id:        secretID,
			version:   2,
			updatedAt: future,
			deleted:   true,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionDelete,
		},
	}
	s := NewClientStrategy()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := s.Rules(tt.id, tt.version, tt.updatedAt, tt.deleted, tt.secretVersions)
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
		deleted        bool
		secretVersions models.SecretVersions
		wantAction     models.SyncAction
	}{
		{
			name:           "секрет отсутствует на сервере",
			id:             secretID,
			version:        1,
			updatedAt:      now,
			deleted:        false,
			secretVersions: models.SecretVersions{},
			wantAction:     models.ActionUpdate,
		},
		{
			name:      "версия клиента новее серверной",
			id:        secretID,
			version:   3,
			updatedAt: now,
			deleted:   false,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 1, UpdatedAt: now},
			},
			wantAction: models.ActionUpdate,
		},
		{
			name:      "версии равны, но время отличается",
			id:        secretID,
			version:   1,
			updatedAt: future,
			deleted:   false,
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
			deleted:   false,
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
			deleted:   false,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 2, UpdatedAt: now},
			},
			wantAction: models.ActionSkip,
		},
		{
			name:           "секрет отсутствует на сервере и удален клиентом",
			id:             secretID,
			version:        1,
			updatedAt:      now,
			deleted:        true,
			secretVersions: models.SecretVersions{},
			wantAction:     models.ActionDelete,
		},
		{
			name:      "версия клиента новее серверной и секрет удален клиентом",
			id:        secretID,
			version:   3,
			updatedAt: now,
			deleted:   true,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 1, UpdatedAt: now},
			},
			wantAction: models.ActionDelete,
		},
		{
			name:      "версии равны, время отличается и секрет удален клиентом",
			id:        secretID,
			version:   1,
			updatedAt: future,
			deleted:   true,
			secretVersions: models.SecretVersions{
				secretID: models.VersionTime{Version: 1, UpdatedAt: now},
			},
			wantAction: models.ActionDelete,
		},
	}
	s := NewServerStrategy()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := s.Rules(tt.id, tt.version, tt.updatedAt, tt.deleted, tt.secretVersions)
			assert.Equal(t, tt.wantAction, action)
		})
	}
}
