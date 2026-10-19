package strategy

import (
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

type ClientStrategy struct{}

func NewClientStrategy() ClientStrategy {
	return ClientStrategy{}
}

func (s ClientStrategy) Rules(id string, version int, updatedAt time.Time, versions models.SecretVersions) models.SyncAction {
	localValue, ok := versions[id]
	if !ok {
		return models.ActionUpdateSecret
	}
	if version > localValue.Version {
		return models.ActionUpdateSecret
	}
	if version == localValue.Version && !updatedAt.Equal(localValue.UpdatedAt) {
		return models.ActionConflict
	}
	return models.ActionSkip
}

type ServerStrategy struct{}

func NewServerStrategy() ServerStrategy {
	return ServerStrategy{}
}

func (s ServerStrategy) Rules(id string, version int, updatedAt time.Time, versions models.SecretVersions) models.SyncAction {
	v, ok := versions[id]
	if !ok {
		return models.ActionUpdateSecret
	}
	if version > v.Version {
		return models.ActionUpdateSecret
	}
	if version == v.Version && !updatedAt.Equal(v.UpdatedAt) {
		return models.ActionConflict
	}
	return models.ActionSkip
}
