package strategy

import (
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type ClientStrategy struct{}

func NewClientStrategy() ClientStrategy {
	return ClientStrategy{}
}

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

type ServerStrategy struct{}

func NewServerStrategy() ServerStrategy {
	return ServerStrategy{}
}

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
