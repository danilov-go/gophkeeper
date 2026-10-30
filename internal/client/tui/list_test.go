package tui

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/danilov-go/gophkeeper/internal/client/tui/mocks"
	"github.com/danilov-go/gophkeeper/internal/models"
)

const (
	secretID1 = "secretID1"
	secretID2 = "secretID2"
)

func TestListModel_Init(t *testing.T) {
	service := mocks.NewSecretService(t)
	secrets := []models.Secret{{ID: "1", Type: models.TypeText}}
	service.On("GetAll", mock.Anything, models.SecretType("text")).Return(secrets, nil).Maybe()
	m := NewListModel(service, models.TypeText)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestListModel_Update(t *testing.T) {
	secrets := []models.Secret{
		{ID: secretID1, Type: models.TypeText},
		{ID: secretID2, Type: models.TypeText},
	}
	tests := []struct {
		name        string
		sType       models.SecretType
		initCursor  int
		secrets     []models.Secret
		msg         tea.Msg
		setupMock   func(s *mocks.SecretService)
		cursor      int
		expMsg      string
		expErr      error
		wantInitCmd bool
	}{
		{
			name:       "нажатие down",
			sType:      models.TypeText,
			initCursor: 0,
			secrets:    secrets,
			msg:        tea.KeyMsg{Type: tea.KeyDown},
			cursor:     1,
		},
		{
			name:       "нажатие up",
			sType:      models.TypeText,
			initCursor: 1,
			secrets:    secrets,
			msg:        tea.KeyMsg{Type: tea.KeyUp},
			cursor:     0,
		},
		{
			name:       "обновление внутренний таймер TOTP",
			sType:      models.TypeLoginPassword,
			initCursor: 0,
			msg:        MsgOTP(time.Now()),
			cursor:     0,
		},
		{
			name:       "обновление данных",
			sType:      models.TypeText,
			initCursor: 0,
			msg:        SyncOccurrenceMsg{},
			setupMock: func(s *mocks.SecretService) {
				s.On("GetAll", mock.Anything, models.SecretType("text")).Return([]models.Secret{}, nil).Maybe()
			},
			cursor:      0,
			expMsg:      "Данные обновлены",
			wantInitCmd: true,
		},
		{
			name:       "нажатие d",
			sType:      models.TypeText,
			initCursor: 1,
			secrets:    secrets,
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
			setupMock: func(s *mocks.SecretService) {
				s.On("Delete", mock.Anything, secretID1).Return(nil).Once()
				s.On("GetAll", mock.Anything, models.SecretType("text")).Return([]models.Secret{}, nil).Maybe()
			},
			cursor:      0,
			wantInitCmd: true,
		},
		{
			name:       "Ошибка метода Delete записывается в поле err модели",
			sType:      models.TypeText,
			initCursor: 0,
			secrets:    secrets,
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
			setupMock: func(s *mocks.SecretService) {
				s.On("Delete", mock.Anything, secretID1).Return(errTest).Once()
			},
			cursor: 0,
			expErr: errTest,
		},
		{
			name:       "Получение нового слайса секретов обновляет данные модели",
			sType:      models.TypeText,
			initCursor: 0,
			msg:        secrets,
			cursor:     0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			if tt.setupMock != nil {
				tt.setupMock(service)
			}
			m := NewListModel(service, tt.sType)
			m.secrets = tt.secrets
			m.cursor = tt.cursor
			msg := tt.msg
			if key, ok := tt.msg.(tea.KeyMsg); ok {
				if string(key.Runes) == "up" || key.Type == tea.KeyUp {
					msg = tea.KeyMsg{Type: tea.KeyUp}
				} else if string(key.Runes) == "down" || key.Type == tea.KeyDown {
					msg = tea.KeyMsg{Type: tea.KeyDown}
				}
			}
			updatedModel, cmd := m.Update(msg)
			model := updatedModel.(ListModel)
			assert.Equal(t, tt.cursor, model.cursor)
			assert.Equal(t, tt.expMsg, model.statusMsg)
			assert.Equal(t, tt.expErr, model.err)
			if tt.wantInitCmd {
				assert.NotNil(t, cmd)
			}
		})
	}
}

func TestListModel_ExportCmd(t *testing.T) {
	service := mocks.NewSecretService(t)
	secret := models.Secret{
		ID:   secretID,
		Type: models.TypeBinary,
		Meta: &models.MetaBinary{Name: "test", Type: ".txt"},
		Data: &models.BinaryData{Data: []byte("test data")},
	}
	service.On("ExportFile", "test.txt", []byte("test data")).Return(nil).Once()
	m := NewListModel(service, models.TypeBinary)
	m.secrets = []models.Secret{secret}
	m.cursor = 0
	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	assert.NotNil(t, cmd)
	msg := cmd()
	assert.Equal(t, MsgList{Text: "Файл успешно сохранен"}, msg)
	model, _ := updatedModel.Update(msg)
	assert.Equal(t, "Файл успешно сохранен", model.(ListModel).statusMsg)
}

func TestListModel_View(t *testing.T) {
	service := mocks.NewSecretService(t)
	tests := []struct {
		name      string
		sType     models.SecretType
		secrets   []models.Secret
		statusMsg string
		err       error
		contains  []string
	}{
		{
			name:     "пустой список заметок",
			sType:    models.TypeText,
			secrets:  []models.Secret{},
			contains: []string{"Категория: text", "Список пуст."},
		},
		{
			name:     "данные отсутствуют на сервере",
			sType:    models.TypeText,
			err:      errors.New("данные отсутствуют на сервере"),
			contains: []string{"Список пуст."},
		},
		{
			name:     "отображение ошибки",
			sType:    models.TypeText,
			err:      errTest,
			contains: []string{"Ошибка: test error"},
		},
		{
			name:      "завершение синхронизации",
			sType:     models.TypeText,
			statusMsg: "Синхронизация завершена",
			contains:  []string{"Синхронизация завершена"},
		},
		{
			name:  "отображение списка TypeCard",
			sType: models.TypeCard,
			secrets: []models.Secret{
				{
					ID:   secretID,
					Type: models.TypeCard,
					Meta: &models.MetaCard{Name: "Name", BankName: "BankName"},
				},
			},
			contains: []string{"> [ID: secretID] Карта: Name [BankName]"},
		},
		{
			name:  "отображение списка TypeLoginPassword с TOTP",
			sType: models.TypeLoginPassword,
			secrets: []models.Secret{
				{
					ID:   secretID,
					Type: models.TypeLoginPassword,
					Meta: &models.MetaLoginPassword{Name: "Name", URL: "URL"},
					Data: &models.LoginPassword{Login: "admin", SecretTOTP: "MZXW6YTBOI======"},
				},
			},
			contains: []string{"[ID: secretID] Аккаунт: Name (URL) | 2FA:"},
		},
		{
			name:  "отображение списка TypeText",
			sType: models.TypeText,
			secrets: []models.Secret{
				{
					ID:   secretID,
					Type: models.TypeText,
					Meta: &models.MetaText{Name: "Name"},
				},
			},
			contains: []string{"Заметка: Name"},
		},
		{
			name:  "отображение списка TypeBinary",
			sType: models.TypeBinary,
			secrets: []models.Secret{
				{
					ID:   secretID,
					Type: models.TypeBinary,
					Meta: &models.MetaBinary{Name: "Name", Type: "test.txt", Size: 123},
				},
			},
			contains: []string{"Файл: Name (test.txt) | 123 байт"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewListModel(service, tt.sType)
			m.secrets = tt.secrets
			m.statusMsg = tt.statusMsg
			m.err = tt.err
			m.cursor = 0
			text := m.View()
			for _, expText := range tt.contains {
				assert.Contains(t, text, expText)
			}
		})
	}
}
