package service

import (
	"context"
	"errors"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// SyncData выполняет синхронизацию данных клиента с сервером.
func (s *ClientService) SyncData(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.RLock()
	offline := s.offline
	hasHash := s.authHash != nil
	s.mu.RUnlock()
	if offline && !hasHash {
		return nil
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
		if cipherData, err = s.syncError(ctx, err, localVersions, changelogs); err != nil {
			return err
		}
		s.mu.RLock()
		offline = s.offline
		s.mu.RUnlock()
		if offline {
			return nil
		}
	}
	var updateSecrets []models.CipherData
	for _, secret := range cipherData {
		switch s.strategy.Rules(secret.ID, secret.Version, secret.UpdatedAt, secret.Deleted, localVersions) {
		case models.ActionConflict:
			if err = s.storage.SaveConflict(ctx, secret); err != nil {
				return err
			}
		case models.ActionUpdate:
			updateSecrets = append(updateSecrets, secret)
		case models.ActionDelete:
			if err = s.storage.DeleteSecret(ctx, secret.ID, true); err != nil {
				return err
			}
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
			if err = s.storage.DeleteChangelog(lg.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ClientService) syncError(ctx context.Context, syncErr error, localVersions map[string]models.VersionTime, changelogs []models.Changelog) ([]models.CipherData, error) {
	if !errors.Is(syncErr, models.ErrNoToken) && !errors.Is(syncErr, models.ErrClientUnauthorized) {
		return nil, syncErr
	}
	s.mu.RLock()
	login := s.login
	hash := s.authHash
	s.mu.RUnlock()
	if login == "" || hash == nil {
		return nil, models.ErrUserNotFound
	}
	authHash := hash.Bytes()
	authErr := s.sender.Auth(ctx, login, string(authHash))
	if authErr != nil {
		s.mu.Lock()
		s.offline = true
		s.mu.Unlock()
		s.sender.Logout()
		return nil, nil
	}
	s.mu.Lock()
	s.offline = false
	s.mu.Unlock()
	cipherData, err := s.sender.Sync(ctx, localVersions, changelogs)
	if err != nil {
		return nil, err
	}
	return cipherData, nil
}

func (s *ClientService) trigger() {
	if s.syncCh != nil {
		select {
		case s.syncCh <- struct{}{}:
		default:
		}
	}
}
