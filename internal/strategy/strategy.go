package strategy

import (
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type ClientStrategy struct{}

func NewClientStrategy() ClientStrategy {
	return ClientStrategy{}
}

func (s ClientStrategy) Rules(id int, serverVersion int, serverUpdatedAt time.Time, localVersions models.SecretVersions) models.SyncAction {
	localValue, exists := localVersions[id]
	if !exists {
		return models.ActionUpdateSecret
	}
	if serverVersion > localValue.Version {
		return models.ActionUpdateSecret
	}
	if serverVersion == localValue.Version && !serverUpdatedAt.Equal(localValue.UpdatedAt) {
		return models.ActionConflict
	}
	return models.ActionSkip
}

type ServerStrategy struct{}

func NewServerStrategy() ServerStrategy {
	return ServerStrategy{}
}

func (s ServerStrategy) Rules(id int, version int, updatedAt time.Time, versions models.SecretVersions) models.SyncAction {
	v, ok := versions[id]
	if !ok {
		return models.ActionConflict
	}
	if version > v.Version {
		return models.ActionConflict
	}
	if version == v.Version && !updatedAt.Equal(v.UpdatedAt) {
		return models.ActionConflict
	}
	return models.ActionSkip
}
