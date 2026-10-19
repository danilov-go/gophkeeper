package service

import (
	"context"
	"errors"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// GetUnresolvedConflicts извлекает все отложенные конфликты и расшифровывает обе версии данных для TUI.
func (s *ClientService) GetConflicts(ctx context.Context) ([]models.Conflict, error) {
	conflicts, err := s.storage.GetConflicts(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	if s.key == nil {
		s.mu.RUnlock()
		return nil, errors.New("отсутствует ключ шифрования")
	}
	key := s.key.Bytes()
	s.mu.RUnlock()
	var result []models.Conflict
	for _, conflict := range conflicts {
		localSecret, err := s.Get(ctx, conflict.ID)
		if err != nil {
			localSecret = models.Secret{ID: conflict.ID, Type: conflict.Type}
		}
		secret, err := s.decryptCipher(key, conflict)
		if err != nil {
			continue
		}
		result = append(result, models.Conflict{
			ID:     conflict.ID,
			Local:  localSecret,
			Server: secret,
		})
	}
	return result, nil
}

// ResolveConflictUI применяет выбранное пользователем в TUI решение к конкретному секрету.
func (s *ClientService) FixConflict(ctx context.Context, id int, keepServer bool) error {
	conflicts, err := s.storage.GetConflicts(ctx)
	if err != nil {
		return err
	}
	var conflict models.CipherData
	ok := false
	for _, sc := range conflicts {
		if sc.ID == id {
			conflict = sc
			ok = true
			break
		}
	}
	if !ok {
		return errors.New("конфликтная запись не найдена")
	}
	if keepServer {
		if err := s.storage.SyncUpdate(ctx, []models.CipherData{conflict}); err != nil {
			return err
		}
	} else {
		secret, err := s.storage.GetSecret(ctx, id)
		if err != nil {
			return err
		}
		secret.Version++
		secret.UpdatedAt = time.Now()
		if err := s.storage.UpdateSecret(ctx, secret); err != nil {
			return err
		}
	}
	if err := s.storage.DeleteConflict(ctx, id); err != nil {
		return err
	}
	s.trigger()
	return nil
}
