// Package strategy содержит стратегии синхронизации данных между клиентом и сервером.
package strategy

import (
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// ClientStrategy определяет стратегию синхронизации на стороне клиента.
type ClientStrategy struct{}

// NewClientStrategy создает новый экземпляр ClientStrategy.
func NewClientStrategy() ClientStrategy {
	return ClientStrategy{}
}

// Rules определяет правила по синхронизации для клиента.
func (s ClientStrategy) Rules(id string, version int, updatedAt time.Time, deleted bool, versions models.SecretVersions) models.SyncAction {
	v, ok := versions[id]
	if !ok {
		if deleted {
			return models.ActionSkip
		}
		return models.ActionUpdate
	}
	if version > v.Version {
		if deleted {
			return models.ActionDelete
		}
		return models.ActionUpdate
	}
	if version == v.Version && !updatedAt.Equal(v.UpdatedAt) {
		if deleted {
			return models.ActionDelete
		}
		return models.ActionConflict
	}
	return models.ActionSkip
}

// ClientStrategy определяет стратегию синхронизации на стороне сервера.
type ServerStrategy struct{}

// NewServerStrategy создает новый экземпляр NewServerStrategy.
func NewServerStrategy() ServerStrategy {
	return ServerStrategy{}
}

// Rules определяет правила по синхронизации для сервера.
func (s ServerStrategy) Rules(id string, version int, updatedAt time.Time, deleted bool, versions models.SecretVersions) models.SyncAction {
	v, ok := versions[id]
	if !ok {
		if deleted {
			return models.ActionDelete
		}
		return models.ActionUpdate
	}
	if version > v.Version {
		if deleted {
			return models.ActionDelete
		}
		return models.ActionUpdate
	}
	if version == v.Version && !updatedAt.Equal(v.UpdatedAt) {
		if deleted {
			return models.ActionDelete
		}
		return models.ActionConflict
	}
	return models.ActionSkip
}
