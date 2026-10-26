package service

import (
	"context"
	"errors"
	"testing"

	"github.com/awnumar/memguard"
	"github.com/danilov-go/gophkeeper/internal/client/service/mocks"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	secretID = "secretID"
)

var testKey = []byte("12345678912345678912345678912345")
var errTest = errors.New("test error")

func TestClientService_GetConflicts(t *testing.T) {
	memguard.Purge()
	storage := mocks.NewStorage(t)
	sender := mocks.NewSender(t)
	strategy := mocks.NewStrategy(t)
	syncCh := make(chan struct{}, 1)
	svc := NewClientService(storage, sender, strategy, syncCh)
	svc.key = memguard.NewBufferFromBytes(testKey)
	cipher, err := svc.encrypt(testKey, &models.MetaCard{Name: "Name"}, &models.Card{Number: "4242"})
	assert.NoError(t, err)
	conflict := models.CipherData{
		ID:     secretID,
		Type:   models.TypeCard,
		Cipher: cipher,
	}
	tests := []struct {
		name      string
		setupKey  bool
		setupMock func(mStorage *mocks.Storage)
		count     int
		expErr    string
	}{
		{
			name:     "положительный тест",
			setupKey: true,
			setupMock: func(mStorage *mocks.Storage) {
				storage.On("GetConflicts", mock.Anything).Return([]models.CipherData{conflict}, nil).Once()
				storage.On("GetSecret", mock.Anything, secretID).Return(conflict, nil).Once()
			},
			count:  1,
			expErr: "",
		},
		{
			name:     "отсутствует ключ шифрования",
			setupKey: false,
			setupMock: func(mStorage *mocks.Storage) {
				storage.On("GetConflicts", mock.Anything).Return([]models.CipherData{conflict}, nil).Once()
			},
			count:  0,
			expErr: "отсутствует ключ шифрования",
		},
		{
			name:     "ошибка  БД",
			setupKey: true,
			setupMock: func(mStorage *mocks.Storage) {
				storage.On("GetConflicts", mock.Anything).Return(nil, errTest).Once()
			},
			count:  0,
			expErr: "test error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.ExpectedCalls = nil
			tt.setupMock(storage)
			if tt.setupKey {
				svc.key = memguard.NewBufferFromBytes(testKey)
			} else {
				svc.key = nil
			}
			conflicts, err := svc.GetConflicts(context.Background())
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				assert.Nil(t, conflicts)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, conflicts, tt.count)
			if tt.count > 0 {
				assert.Equal(t, secretID, conflicts[0].ID)
				assert.Equal(t, secretID, conflicts[0].Local.ID)
				assert.Equal(t, secretID, conflicts[0].Server.ID)
			}
		})
	}
}

func TestClientService_FixConflict(t *testing.T) {
	memguard.Purge()
	conflict := models.CipherData{ID: secretID, Type: models.TypeCard, Version: 1}
	tests := []struct {
		name       string
		id         string
		keepServer bool
		setupMock  func(mStorage *mocks.Storage)
		expErr     string
	}{
		{
			name:       "положительный тест (keepServer = true)",
			id:         secretID,
			keepServer: true,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("GetConflicts", mock.Anything).Return([]models.CipherData{conflict}, nil).Once()
				mStorage.On("SyncUpdate", mock.Anything, []models.CipherData{conflict}).Return(nil).Once()
				mStorage.On("DeleteConflict", mock.Anything, secretID).Return(nil).Once()
			},
			expErr: "",
		},
		{
			name:       "положительный тест (keepServer = false)",
			id:         secretID,
			keepServer: false,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("GetConflicts", mock.Anything).Return([]models.CipherData{conflict}, nil).Once()
				mStorage.On("GetSecret", mock.Anything, secretID).
					Return(models.CipherData{ID: secretID, Version: 3}, nil).Once()
				mStorage.On("UpdateSecret", mock.Anything, mock.MatchedBy(func(cd models.CipherData) bool {
					return cd.ID == secretID && cd.Version == 4
				})).Return(nil).Once()
				mStorage.On("DeleteConflict", mock.Anything, secretID).Return(nil).Once()
			},
			expErr: "",
		},
		{
			name:       "запись не найдена",
			id:         "unknow",
			keepServer: true,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("GetConflicts", mock.Anything).Return([]models.CipherData{conflict}, nil).Once()
			},
			expErr: "конфликтная запись не найдена",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			tt.setupMock(storage)
			svc := NewClientService(storage, sender, strategy, syncCh)
			err := svc.FixConflict(context.Background(), tt.id, tt.keepServer)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, syncCh, 1)
		})
	}
}
