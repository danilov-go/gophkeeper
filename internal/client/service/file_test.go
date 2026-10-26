package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/awnumar/memguard"
	"github.com/danilov-go/gophkeeper/internal/client/service/mocks"
	"github.com/stretchr/testify/assert"
)

func TestClientService_GetFile(t *testing.T) {
	storage := mocks.NewStorage(t)
	sender := mocks.NewSender(t)
	strategy := mocks.NewStrategy(t)
	syncCh := make(chan struct{}, 1)
	svc := NewClientService(storage, sender, strategy, syncCh)
	testDir := t.TempDir()
	file := filepath.Join(testDir, "test.txt")
	secretData := []byte("secret data")
	err := os.WriteFile(file, secretData, 0600)
	assert.NoError(t, err)
	tests := []struct {
		name    string
		path    string
		want    []byte
		wantErr bool
	}{
		{
			name:    "положительный тест",
			path:    file,
			want:    secretData,
			wantErr: false,
		},
		{
			name:    "файла не существует",
			path:    filepath.Join(testDir, "unknow.txt"),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "передана пустая строка",
			path:    "",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := svc.GetFile(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, data)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, data)
		})
	}
}

func TestClientService_ExportFile(t *testing.T) {
	storage := mocks.NewStorage(t)
	sender := mocks.NewSender(t)
	strategy := mocks.NewStrategy(t)
	syncCh := make(chan struct{}, 1)
	svc := NewClientService(storage, sender, strategy, syncCh)
	testDir := t.TempDir()
	file := filepath.Join(testDir, "test.txt")
	secretData := []byte("secret data")
	tests := []struct {
		name    string
		path    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "успешный экспорт файла",
			path:    file,
			data:    secretData,
			wantErr: false,
		},
		{
			name:    "запись в директорию при пустом пути",
			path:    "",
			data:    secretData,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ExportFile(tt.path, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			data, err := os.ReadFile(tt.path)
			assert.NoError(t, err)
			assert.Equal(t, tt.data, data)
		})
	}
}

func TestClientService_Logout(t *testing.T) {
	memguard.Purge()
	tests := []struct {
		name      string
		setupKey  bool
		setupMock func(mSender *mocks.Sender)
	}{
		{
			name:     "положительный тест",
			setupKey: true,
			setupMock: func(mSender *mocks.Sender) {
				mSender.On("Logout").Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := mocks.NewStorage(t)
			sender := mocks.NewSender(t)
			strategy := mocks.NewStrategy(t)
			syncCh := make(chan struct{}, 1)
			tt.setupMock(sender)
			svc := NewClientService(storage, sender, strategy, syncCh)
			if tt.setupKey {
				svc.key = memguard.NewBufferFromBytes([]byte("test secret key"))
			}
			svc.Logout()
			assert.Nil(t, svc.key)
			sender.AssertExpectations(t)
		})
	}
}
