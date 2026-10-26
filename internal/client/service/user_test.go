package service

import (
	"context"
	"errors"
	"testing"

	"github.com/danilov-go/gophkeeper/internal/client/service/mocks"
	"github.com/danilov-go/gophkeeper/internal/crypto"

	"github.com/awnumar/memguard"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	login    = "login"
	password = "password"
)

func TestClientService_Register(t *testing.T) {
	memguard.Purge()
	tests := []struct {
		name          string
		login         string
		password      string
		cancelContext bool
		setupMock     func(mStorage *mocks.Storage, mSender *mocks.Sender)
		expErr        string
	}{
		{
			name:     "положительный тест",
			login:    login,
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("Register", mock.Anything, login, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil).Once()
				mStorage.On("SaveUser", mock.Anything, login, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil).Once()
			},
			expErr: "",
		},
		{
			name:      "пустой логин",
			login:     "",
			password:  password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {},
			expErr:    "пустой логин или пароль",
		},
		{
			name:     "ошибка регистрации на сервере",
			login:    login,
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("Register", mock.Anything, login, mock.Anything, mock.Anything).
					Return(errTest).Once()
			},
			expErr: errTest.Error(),
		},
		{
			name:     "ошибка локального сохранения",
			login:    login,
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("Register", mock.Anything, login, mock.Anything, mock.Anything).
					Return(nil).Once()
				mStorage.On("SaveUser", mock.Anything, login, mock.Anything, mock.Anything).
					Return(errTest).Once()
			},
			expErr: errTest.Error(),
		},
		{
			name:          "отмена контекста",
			login:         login,
			password:      password,
			cancelContext: true,
			setupMock:     func(mStorage *mocks.Storage, mSender *mocks.Sender) {},
			expErr:        "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			tt.setupMock(storage, sender)
			svc := NewClientService(storage, sender, strategy, syncCh)
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := svc.Register(ctx, tt.login, tt.password)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestClientService_Login(t *testing.T) {
	memguard.Purge()
	salt, err := crypto.GenerateSalt()
	assert.NoError(t, err)
	_, hash, err := crypto.GenerateKey(salt, password)
	assert.NoError(t, err)
	tests := []struct {
		name          string
		login         string
		password      string
		cancelContext bool
		setupMock     func(mStorage *mocks.Storage, mSender *mocks.Sender)
		expErr        string
	}{
		{
			name:     "положительный тест (онлайн)",
			login:    login,
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("GetSalt", mock.Anything, login).Return(salt, nil).Once()
				mSender.On("Auth", mock.Anything, login, mock.AnythingOfType("string")).Return(nil).Once()
				mStorage.On("SaveUser", mock.Anything, login, mock.Anything, salt).Return(nil).Once()
			},
			expErr: "",
		},
		{
			name:      "пустой пароль",
			login:     login,
			password:  "",
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {},
			expErr:    "пустой логин или пароль",
		},
		{
			name:     "положительный тест (оффлайн)",
			login:    login,
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("GetSalt", mock.Anything, login).Return("", errTest).Once()
				mStorage.On("GetUser", mock.Anything, login).Return(models.LocalUser{
					Salt: salt,
					Hash: hash,
				}, nil).Once()
			},
			expErr: "",
		},
		{
			name:     "неверный пароль",
			login:    login,
			password: "wrong_password",
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("GetSalt", mock.Anything, login).Return("", errTest).Once()
				mStorage.On("GetUser", mock.Anything, login).Return(models.LocalUser{
					Salt: salt,
					Hash: hash,
				}, nil).Once()
			},
			expErr: "неверный пароль",
		},
		{
			name:     "пользователь отсутствует в локальной БД",
			login:    "unknown",
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("GetSalt", mock.Anything, "unknown").Return("", errTest).Once()
				mStorage.On("GetUser", mock.Anything, "unknown").Return(models.LocalUser{}, errors.New("not found")).Once()
			},
			expErr: "пользователь не найден",
		},
		{
			name:     "неверные данные на сервере",
			login:    login,
			password: password,
			setupMock: func(mStorage *mocks.Storage, mSender *mocks.Sender) {
				mSender.On("GetSalt", mock.Anything, login).Return(salt, nil).Once()
				mSender.On("Auth", mock.Anything, login, mock.Anything).Return(errTest).Once()
			},
			expErr: errTest.Error(),
		},
		{
			name:          "ошибка отмены контекста",
			login:         login,
			password:      password,
			cancelContext: true,
			setupMock:     func(mStorage *mocks.Storage, mSender *mocks.Sender) {},
			expErr:        "context canceled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			tt.setupMock(storage, sender)
			svc := NewClientService(storage, sender, strategy, syncCh)
			ctx := t.Context()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			err := svc.Login(ctx, tt.login, tt.password)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
