package service

import (
	"context"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func (s *ClientService) SyncData(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	localVersions, err := s.storage.GetVersion(ctx)
	if err != nil {
		return err
	}
	changelogs, err := s.storage.GetChangelog()
	if err != nil {
		return err
	}
	cipherData, err := s.sender.Sync(ctx, localVersions, changelogs)
	if err != nil {
		return err
	}
	var updateSecrets []models.CipherData
	for _, secret := range cipherData {
		switch s.strategy.Rules(secret.ID, secret.Version, secret.UpdatedAt, secret.Deleted, localVersions) {
		case models.ActionConflict:
			_ = s.storage.SaveConflict(ctx, secret)
		case models.ActionUpdate:
			updateSecrets = append(updateSecrets, secret)
		case models.ActionDelete:
			_ = s.storage.DeleteSecret(ctx, secret.ID, true)
		case models.ActionSkip:
			continue
		}
	}
	if len(updateSecrets) > 0 {
		err = s.storage.SyncUpdate(ctx, updateSecrets)
		if err != nil {
			return err
		}
	}
	if len(changelogs) > 0 {
		for _, lg := range changelogs {
			_ = s.storage.DeleteChangelog(lg.ID)
		}
	}
	return nil
}

func (s *ClientService) trigger() {
	if s.syncCh != nil {
		select {
		case s.syncCh <- struct{}{}:
		default:
		}
	}
}
