package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/danilov-go/gophkeeper/internal/client/tui/mocks"
	"github.com/danilov-go/gophkeeper/internal/models"
)

const (
	secretID = "secretID"
)

func TestFormModel_Init(t *testing.T) {
	service := mocks.NewSecretService(t)
	m := NewFormModel(service, models.TypeText)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestFormModel_NewFormModel(t *testing.T) {
	service := mocks.NewSecretService(t)
	types := []models.SecretType{
		models.TypeCard,
		models.TypeLoginPassword,
		models.TypeText,
		models.TypeBinary,
	}
	for _, sType := range types {
		t.Run(string(sType), func(t *testing.T) {
			m := NewFormModel(service, sType)
			assert.Equal(t, sType, m.sType)
			assert.Empty(t, m.secretID)
			assert.NotEmpty(t, m.option)
			assert.NotEmpty(t, m.inputs)
			assert.True(t, m.inputs[0].Focused())
		})
	}
}

func TestFormModel_NewEditForm(t *testing.T) {
	service := mocks.NewSecretService(t)
	tests := []struct {
		name   string
		secret models.Secret
	}{
		{
			name: "заполнение формы (TypeCard)",
			secret: models.Secret{
				ID:   secretID,
				Type: models.TypeCard,
				Meta: &models.MetaCard{Name: "Name", BankName: "BankName"},
				Data: &models.Card{Number: "4242", Date: "11/26", UserName: "UserName", Key: "123"},
			},
		},
		{
			name: "заполнение формы (TypeLoginPassword)",
			secret: models.Secret{
				ID:   secretID,
				Type: models.TypeLoginPassword,
				Meta: &models.MetaLoginPassword{Name: "Name", URL: "URL"},
				Data: &models.LoginPassword{Login: "Login", Password: "Password", SecretTOTP: "totp"},
			},
		},
		{
			name: "заполнение формы (TypeText)",
			secret: models.Secret{
				ID:   secretID,
				Type: models.TypeText,
				Meta: &models.MetaText{Name: "Name"},
				Data: &models.TextData{Text: "Text"},
			},
		},
		{
			name: "заполнение формы (TypeBinary)",
			secret: models.Secret{
				ID:   secretID,
				Type: models.TypeBinary,
				Meta: &models.MetaBinary{Name: "test.txt", Size: 123, Type: ".txt"},
				Data: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewEditForm(service, tt.secret)
			assert.Equal(t, tt.secret.Type, m.sType)
			assert.Equal(t, tt.secret.ID, m.secretID)
			assert.NotEmpty(t, m.inputs)
			assert.NotEmpty(t, m.inputs[0].Value())
		})
	}
}

func TestFormModel_Update(t *testing.T) {
	service := mocks.NewSecretService(t)
	tests := []struct {
		name        string
		initFocus   int
		msg         tea.Msg
		setupMock   func(s *mocks.SecretService)
		focus       int
		wantSaved   bool
		expErr      error
		wantSaveCmd bool
	}{
		{

			name:      "нажатие down",
			initFocus: 0,
			msg:       tea.KeyMsg{Type: tea.KeyDown},
			focus:     1,
		},
		{
			name:      "нажатие up",
			initFocus: 1,
			msg:       tea.KeyMsg{Type: tea.KeyUp},
			focus:     0,
		},
		{
			name:        "нажатие enter",
			initFocus:   1,
			msg:         tea.KeyMsg{Type: tea.KeyEnter},
			focus:       1,
			wantSaveCmd: true,
		},
		{
			name:      "получение строки success",
			initFocus: 0,
			msg:       "success",
			focus:     0,
			wantSaved: true,
		},
		{
			name:      "получение ошибки",
			initFocus: 0,
			msg:       errTest,
			focus:     0,
			expErr:    errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(service)
			}
			m := NewFormModel(service, models.TypeText)
			m.focus = tt.initFocus
			currentMsg := tt.msg
			if key, ok := tt.msg.(tea.KeyMsg); ok {
				if string(key.Runes) == "up" || key.Type == tea.KeyUp {
					currentMsg = tea.KeyMsg{Type: tea.KeyUp}
				} else if string(key.Runes) == "down" || key.Type == tea.KeyDown {
					currentMsg = tea.KeyMsg{Type: tea.KeyDown}
				}
			}
			updatedModel, cmd := m.Update(currentMsg)
			model := updatedModel.(FormModel)
			assert.Equal(t, tt.focus, model.focus)
			assert.Equal(t, tt.wantSaved, model.saved)
			assert.Equal(t, tt.expErr, model.err)
			if tt.wantSaveCmd {
				assert.NotNil(t, cmd)
			}
		})
	}
}

func TestFormModel_SwitchFocus_BinarySpecialRules(t *testing.T) {
	service := mocks.NewSecretService(t)
	tests := []struct {
		name      string
		initFocus int
		focus     int
	}{
		{
			name:      "пропуск поля размера и типа (TypeBinary)",
			initFocus: 1,
			focus:     3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewFormModel(service, models.TypeBinary)
			m.focus = tt.initFocus
			updatedModel, _ := m.switchFocus()
			assert.Equal(t, tt.focus, updatedModel.focus)
		})
	}
}

func TestFormModel_SaveCmd(t *testing.T) {
	tests := []struct {
		name        string
		sType       models.SecretType
		secretID    string
		inputValues []string
		setupMock   func(s *mocks.SecretService)
		expectedMsg interface{}
	}{
		{
			name:        "положительный тест (Save)",
			sType:       models.TypeText,
			inputValues: []string{"Name", "Text"},
			setupMock: func(s *mocks.SecretService) {
				s.On("Save", mock.Anything,
					&models.MetaText{Name: "Name"},
					&models.TextData{Text: "Text"},
				).Return(secretID, nil).Once()
			},
			expectedMsg: "success",
		},
		{
			name:        "положительный тест (Update)",
			sType:       models.TypeCard,
			secretID:    secretID,
			inputValues: []string{"Name", "BankName", "4242", "11/26", "UserName", "123"},
			setupMock: func(s *mocks.SecretService) {
				s.On("Update", mock.Anything, secretID,
					&models.MetaCard{Name: "Name", BankName: "BankName"},
					&models.Card{Number: "4242", Date: "11/26", UserName: "UserName", Key: "123"},
				).Return(nil).Once()
			},
			expectedMsg: "success",
		},
		{
			name:        "положительный тест (TypeBinary)",
			sType:       models.TypeBinary,
			inputValues: []string{"test.txt", "", "", "/test.txt"},
			setupMock: func(s *mocks.SecretService) {
				s.On("GetFile", "/test.txt").Return([]byte("test data"), nil).Once()
				s.On("Save", mock.Anything,
					&models.MetaBinary{Name: "test.txt", Size: 9, Type: ".txt"},
					&models.BinaryData{Data: []byte("test data")},
				).Return(secretID, nil).Once()
			},
			expectedMsg: "success",
		},
		{
			name:        "пустое описание",
			sType:       models.TypeText,
			inputValues: []string{"", "test"},
			expectedMsg: errors.New("добавьте описание"),
		},
		{
			name:        "пустой путь к файлу при создании новой записи",
			sType:       models.TypeBinary,
			inputValues: []string{"File Name", "", "", ""},
			expectedMsg: errors.New("путь к файлу не может быть пустым"),
		},
		{
			name:        "ошибка GetFile",
			sType:       models.TypeBinary,
			inputValues: []string{"unknow.png", "", "", "/invalid/unknow.png"},
			setupMock: func(s *mocks.SecretService) {
				s.On("GetFile", "/invalid/unknow.png").Return(nil, errTest).Once()
			},
			expectedMsg: errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			if tt.setupMock != nil {
				tt.setupMock(service)
			}
			m := NewFormModel(service, tt.sType)
			m.secretID = tt.secretID
			for i, val := range tt.inputValues {
				if i < len(m.inputs) {
					m.inputs[i].SetValue(val)
				}
			}
			resMsg := m.saveCmd()()
			assert.Equal(t, tt.expectedMsg, resMsg)
		})
	}
}

func TestFormModel_View(t *testing.T) {
	service := mocks.NewSecretService(t)
	tests := []struct {
		name     string
		sType    models.SecretType
		secretID string
		saved    bool
		err      error
		text     []string
	}{
		{
			name:  "создание новой записи",
			sType: models.TypeText,
			text:  []string{"Добавление записи: text", "Описание:", "Текст:"},
		},
		{
			name:     "редактирование существующей записи",
			sType:    models.TypeCard,
			secretID: secretID,
			text:     []string{"Редактирование записи [ID: secretID]: card"},
		},
		{
			name:  "успешное сохранение",
			sType: models.TypeText,
			saved: true,
			text:  []string{"Запись сохранена!"},
		},
		{
			name:  "отображение ошибки",
			sType: models.TypeText,
			err:   errTest,
			text:  []string{"Ошибка: test error"},
		},
		{
			name:  "отображение TypeBinary",
			sType: models.TypeBinary,
			text: []string{
				"Название:",
				"Размер:",
				"Тип:",
				"Путь к файлу на диске:",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewFormModel(service, tt.sType)
			m.secretID = tt.secretID
			m.saved = tt.saved
			m.err = tt.err
			text := m.View()
			for _, expText := range tt.text {
				assert.Contains(t, text, expText)
			}
		})
	}
}
