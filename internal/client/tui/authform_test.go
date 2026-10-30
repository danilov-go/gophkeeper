package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/client/tui/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	login    = "login"
	password = "password"
)

var errTest = errors.New("test error")

func TestAuthFormModel_Init(t *testing.T) {
	service := mocks.NewSecretService(t)
	m := NewAuthFormModel(service, false)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestAuthFormModel_Update(t *testing.T) {
	tests := []struct {
		name             string
		wantRegistration bool
		login            string
		password         string
		setupMock        func(s *mocks.SecretService)
		expMsg           tea.Msg
	}{
		{
			name:             "положительный тест (регистрация)",
			wantRegistration: true,
			login:            login,
			password:         password,
			setupMock: func(s *mocks.SecretService) {
				s.On("Register", mock.Anything, login, password).Return(nil).Once()
			},
			expMsg: "auth_success",
		},
		{
			name:             "положительный тест (авторизация)",
			wantRegistration: false,
			login:            login,
			password:         password,
			setupMock: func(s *mocks.SecretService) {
				s.On("Login", mock.Anything, login, password).Return(nil).Once()
			},
			expMsg: "auth_success",
		},
		{
			name:             "пустые полях формы",
			wantRegistration: false,
			login:            "",
			password:         "",
			setupMock:        nil,
			expMsg:           errors.New("поля не могут быть пустыми"),
		},

		{
			name:             "ошибка при входе",
			wantRegistration: false,
			login:            login,
			password:         password,
			setupMock: func(s *mocks.SecretService) {
				s.On("Login", mock.Anything, login, password).Return(errTest).Once()
			},
			expMsg: errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			if tt.setupMock != nil {
				tt.setupMock(service)
			}
			m := NewAuthFormModel(service, tt.wantRegistration)
			m.inputs[0].SetValue(tt.login)
			m.inputs[1].SetValue(tt.password)
			m.focus = 1
			updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			assert.NotNil(t, cmd)
			msg := cmd()
			assert.Equal(t, tt.expMsg, msg)
			model, cmd := updatedModel.Update(msg)
			assert.Nil(t, cmd)
			form := model.(AuthFormModel)
			if strMsg, ok := msg.(string); ok && strMsg == "auth_success" {
				assert.True(t, form.success)
			} else if errMsg, ok := msg.(error); ok {
				assert.Equal(t, errMsg, form.err)
			}
		})
	}
}

func TestAuthFormModel_View(t *testing.T) {
	service := mocks.NewSecretService(t)
	tests := []struct {
		name             string
		wantRegistration bool
		err              error
		success          bool
		text             []string
	}{
		{
			name:             "отображение экрана формы авторизации",
			wantRegistration: false,
			err:              nil,
			success:          false,
			text:             []string{"Авторизация", "Логин ", "Пароль"},
		},
		{
			name:             "отображение экрана формы регистрации",
			wantRegistration: true,
			err:              nil,
			success:          false,
			text:             []string{"Регистрация", "Логин ", "Пароль"},
		},
		{
			name:             "отображение возникшей ошибки",
			wantRegistration: false,
			err:              errors.New("connection failed"),
			success:          false,
			text:             []string{"Ошибка: connection failed"},
		},
		{
			name:             "отображение экрана успешного завершения",
			wantRegistration: false,
			err:              nil,
			success:          true,
			text:             []string{"Успешно!"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewAuthFormModel(service, tt.wantRegistration)
			m.err = tt.err
			m.success = tt.success
			text := m.View()
			for _, expText := range tt.text {
				assert.Contains(t, text, expText)
			}
		})
	}
}
