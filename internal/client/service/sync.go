package service

import "context"

func (s *ClientService) SyncData(ctx context.Context) error {
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
	if len(cipherData) > 0 {
		err = s.storage.SyncUpdate(ctx, cipherData)
		if err != nil {
			return err
		}
	}
	if len(changelogs) > 0 {
		lastLogID := changelogs[len(changelogs)-1].ID
		_ = s.storage.DeleteChangelog(lastLogID)
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
