package service

import (
	"context"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/client/service/mocks"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestClientService_SyncData(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	versions := models.SecretVersions{
		secretID:  {Version: 1, UpdatedAt: now},
		secretID1: {Version: 2, UpdatedAt: now},
		secretID2: {Version: 3, UpdatedAt: now},
	}
	changelogs := []models.Changelog{
		{ID: 5, SecretID: secretID},
	}
	cipherData := []models.CipherData{
		{ID: secretID, Version: 2, UpdatedAt: now, Deleted: false, Cipher: []byte("conflict")},
		{ID: secretID1, Version: 3, UpdatedAt: now, Deleted: false, Cipher: []byte("update")},
		{ID: secretID2, Version: 4, UpdatedAt: now, Deleted: true},
		{ID: secretID3, Version: 1, UpdatedAt: now, Deleted: false, Cipher: []byte("skip")},
	}
	tests := []struct {
		name          string
		cancelContext bool
		setupMock     func(mStorage *mocks.Storage, mSender *mocks.Sender, mStrategy *mocks.Strategy)
		wantErr       bool
	}{
		{
			name: "положительный тест",
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender, mStrategy *mocks.Strategy) {
				mStorage.On("GetVersion", mock.Anything).Return(map[string]models.VersionTime(versions), nil).Once()
				mStorage.On("GetChangelog").Return(changelogs, nil).Once()
				mSender.On("Sync", mock.Anything, map[string]models.VersionTime(versions), changelogs).Return(cipherData, nil).Once()
				mStrategy.On("Rules", secretID, 2, now, false, versions).Return(models.ActionConflict).Once()
				mStrategy.On("Rules", secretID1, 3, now, false, versions).Return(models.ActionUpdate).Once()
				mStrategy.On("Rules", secretID2, 4, now, true, versions).Return(models.ActionDelete).Once()
				mStrategy.On("Rules", secretID3, 1, now, false, versions).Return(models.ActionSkip).Once()
				mStorage.On("SaveConflict", mock.Anything, cipherData[0]).Return(nil).Once()
				mStorage.On("DeleteSecret", mock.Anything, secretID2, true).Return(nil).Once()
				expectedUpdates := []models.CipherData{cipherData[1]}
				mStorage.On("SyncUpdate", mock.Anything, expectedUpdates).Return(nil).Once()
				mStorage.On("DeleteChangelog", 5).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "ошибка при получении локальных версий",
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender, mStrategy *mocks.Strategy) {
				mStorage.On("GetVersion", mock.Anything).Return(nil, errTest).Once()
			},
			wantErr: true,
		},
		{
			name: "ошибка отправки на server",
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender, mStrategy *mocks.Strategy) {
				mStorage.On("GetVersion", mock.Anything).Return(map[string]models.VersionTime(versions), nil).Once()
				mStorage.On("GetChangelog").Return(changelogs, nil).Once()
				mSender.On("Sync", mock.Anything, map[string]models.VersionTime(versions), changelogs).Return(nil, errTest).Once()
			},
			wantErr: true,
		},
		{
			name:          "отмена контекста",
			cancelContext: true,
			setupMock:     func(mStorage *mocks.Storage, mSender *mocks.Sender, mStrategy *mocks.Strategy) {},
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			tt.setupMock(storage, sender, strategy)
			svc := NewClientService(storage, sender, strategy, syncCh)
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := svc.SyncData(ctx)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
