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
	secretID1 = "secretID1"
	secretID2 = "secretID2"
	secretID3 = "secretID3"
)

func TestClientService_Save(t *testing.T) {
	memguard.Purge()
	tests := []struct {
		name      string
		setupKey  bool
		setupMock func(mStorage *mocks.Storage)
		expErr    string
	}{
		{
			name:     "положительный тест",
			setupKey: true,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("SaveSecret", mock.Anything, mock.MatchedBy(func(cd models.CipherData) bool {
					return cd.Type == models.TypeCard && len(cd.Cipher) > 0
				})).Return(secretID, nil).Once()
			},
			expErr: "",
		},
		{
			name:      "отсутствует ключ шифрования",
			setupKey:  false,
			setupMock: func(mStorage *mocks.Storage) {},
			expErr:    "отсутствует ключ шифрования",
		},
		{
			name:     "ошибка сохранения в базу данных",
			setupKey: true,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("SaveSecret", mock.Anything, mock.Anything).Return("", errTest).Once()
			},
			expErr: "test error",
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
			if tt.setupKey {
				svc.key = memguard.NewBufferFromBytes(testKey)
			}
			meta := &models.MetaCard{
				Name: "Name",
			}
			secret := &models.Card{
				Number: "4242",
			}
			id, err := svc.Save(context.Background(), meta, secret)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				assert.Empty(t, id)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, secretID, id)
			assert.Len(t, syncCh, 1)
		})
	}
}

func TestClientService_Update(t *testing.T) {
	memguard.Purge()
	tests := []struct {
		name      string
		id        string
		setupKey  bool
		setupMock func(mStorage *mocks.Storage)
		expErr    string
	}{
		{
			name:     "положительный тест",
			id:       secretID,
			setupKey: true,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("GetSecret", mock.Anything, secretID).
					Return(models.CipherData{ID: secretID, Version: 5}, nil).Once()
				mStorage.On("UpdateSecret", mock.Anything, mock.MatchedBy(func(cd models.CipherData) bool {
					return cd.ID == secretID && cd.Version == 5
				})).Return(nil).Once()
			},
			expErr: "",
		},
		{
			name:     "секрет не найден",
			id:       "unknown",
			setupKey: true,
			setupMock: func(mStorage *mocks.Storage) {
				mStorage.On("GetSecret", mock.Anything, "unknown").
					Return(models.CipherData{}, errors.New(errTest.Error())).Once()
			},
			expErr: errTest.Error(),
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
			if tt.setupKey {
				svc.key = memguard.NewBufferFromBytes(testKey)
			}
			meta := &models.MetaCard{
				Name: "Name",
			}
			secret := &models.Card{
				Number: "4242",
			}
			err := svc.Update(context.Background(), tt.id, meta, secret)
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

func TestClientService_Delete(t *testing.T) {
	storage := mocks.NewStorage(t)
	sender := mocks.NewSender(t)
	strategy := mocks.NewStrategy(t)
	syncCh := make(chan struct{}, 1)
	storage.On("DeleteSecret", mock.Anything, secretID, false).Return(nil).Once()
	svc := NewClientService(storage, sender, strategy, syncCh)
	err := svc.Delete(context.Background(), secretID)
	assert.NoError(t, err)
	assert.Len(t, syncCh, 1)
}

func TestClientService_Get(t *testing.T) {
	memguard.Purge()
	tests := []struct {
		name      string
		id        string
		setupKey  bool
		setupMock func(t *testing.T, mStorage *mocks.Storage, svc *ClientService)
		expErr    string
	}{
		{
			name:     "положительный тест",
			id:       secretID,
			setupKey: true,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				cipher, err := svc.encrypt(testKey, &models.MetaCard{Name: "Name"}, &models.Card{Number: "4242"})
				assert.NoError(t, err)
				mStorage.On("GetSecret", mock.Anything, secretID).Return(models.CipherData{
					ID:     secretID,
					Type:   models.TypeCard,
					Cipher: cipher,
				}, nil).Once()
			},
			expErr: "",
		},
		{
			name:     "ошибка дешифрования",
			id:       secretID,
			setupKey: true,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				mStorage.On("GetSecret", mock.Anything, secretID).Return(models.CipherData{
					ID:     secretID,
					Type:   models.TypeCard,
					Cipher: []byte("invalid"),
				}, nil).Once()
			},
			expErr: "unexpected EOF",
		},
		{
			name:     "секрет не найден в хранилище",
			id:       "unknow",
			setupKey: true,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				mStorage.On("GetSecret", mock.Anything, "unknow").
					Return(models.CipherData{}, errTest).Once()
			},
			expErr: errTest.Error(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			svc := NewClientService(storage, sender, strategy, syncCh)
			if tt.setupKey {
				svc.key = memguard.NewBufferFromBytes(testKey)
			}
			tt.setupMock(t, storage, svc)
			secret, err := svc.Get(context.Background(), tt.id)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.id, secret.ID)
			assert.Equal(t, models.TypeCard, secret.Type)
		})
	}
}

func TestClientService_GetAll(t *testing.T) {
	memguard.Purge()
	tests := []struct {
		name       string
		secretType models.SecretType
		setupKey   bool
		setupMock  func(t *testing.T, mStorage *mocks.Storage, svc *ClientService)
		expErr     string
		wantLen    int
	}{
		{
			name:       "положительный тест",
			secretType: models.TypeCard,
			setupKey:   true,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				cipher, err := svc.encrypt(testKey, &models.MetaCard{Name: "Name"}, &models.Card{Number: "4242"})
				assert.NoError(t, err)
				mStorage.On("GetAllSecret", mock.Anything, models.TypeCard).Return([]models.CipherData{
					{ID: secretID1, Type: models.TypeCard, Cipher: cipher},
					{ID: secretID2, Type: models.TypeCard, Cipher: cipher},
				}, nil).Once()
			},
			expErr:  "",
			wantLen: 2,
		},
		{
			name:       "ошибка дешифрования одного из секретов",
			secretType: models.TypeCard,
			setupKey:   true,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				mStorage.On("GetAllSecret", mock.Anything, models.TypeCard).Return([]models.CipherData{
					{ID: "sec-corrupted", Type: models.TypeCard, Cipher: []byte("invalid")},
				}, nil).Once()
			},
			expErr:  "unexpected EOF",
			wantLen: 0,
		},
		{
			name:       "ошибка получения секретов",
			secretType: models.TypeCard,
			setupKey:   true,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				mStorage.On("GetAllSecret", mock.Anything, models.TypeCard).
					Return(nil, errTest).Once()
			},
			expErr:  errTest.Error(),
			wantLen: 0,
		},
		{
			name:       "отсутствует ключ шифрования",
			secretType: models.TypeCard,
			setupKey:   false,
			setupMock: func(t *testing.T, mStorage *mocks.Storage, svc *ClientService) {
				mStorage.On("GetAllSecret", mock.Anything, models.TypeCard).Return([]models.CipherData{
					{ID: secretID, Type: models.TypeCard, Cipher: []byte("test")},
				}, nil).Once()
			},
			expErr:  "отсутствует ключ шифрования",
			wantLen: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			svc := NewClientService(storage, sender, strategy, syncCh)
			if tt.setupKey {
				svc.key = memguard.NewBufferFromBytes(testKey)
			}
			tt.setupMock(t, storage, svc)
			secrets, err := svc.GetAll(context.Background(), tt.secretType)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				assert.Nil(t, secrets)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, secrets)
			assert.Len(t, secrets, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, secretID1, secrets[0].ID)
				assert.Equal(t, models.TypeCard, secrets[0].Type)
			}
		})
	}
}
