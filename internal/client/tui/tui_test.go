package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/client/tui/mocks"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAppModel_Update(t *testing.T) {
	syncChan := make(chan struct{}, 1)
	occurrenceChan := make(chan struct{}, 1)
	tests := []struct {
		name       string
		state      sessionState
		setupModel func(m *AppModel)
		setupMock  func(m *mocks.SecretService)
		msg        tea.Msg
		expState   sessionState
		wantQuit   bool
	}{
		{
			name:       "выход по Ctrl+C",
			state:      authState,
			setupModel: nil,
			msg: tea.KeyMsg{
				Type:  tea.KeyCtrlC,
				Runes: []rune{},
			},
			expState: authState,
			wantQuit: true,
		},
		{
			name:  "переход в authFormState при выборе пункта меню",
			state: authState,
			setupModel: func(m *AppModel) {
				m.authMenu.chosen = true
				m.authMenu.isRegistration = false
			},
			msg:      tea.KeyMsg{Type: tea.KeyEnter},
			expState: authFormState,
		},
		{
			name:     "возврат из authFormState в authState по нажатию Esc",
			state:    authFormState,
			msg:      tea.KeyMsg{Type: tea.KeyEsc},
			expState: authState,
		},
		{
			name:  "успешная авторизация",
			state: authFormState,
			setupModel: func(m *AppModel) {
				m.authForm.success = true
			},
			msg:      tea.KeyMsg{Type: tea.KeyEnter},
			expState: menuState,
		},
		{
			name:  "нажатие q в menuState",
			state: menuState,
			setupMock: func(m *mocks.SecretService) {
				m.On("Logout").Once()
			},
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			expState: authState,
		},
		{
			name:     "нажатие c",
			state:    menuState,
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}},
			expState: conflictState,
		},
		{
			name:     "Возврат из conflictState в menuState по Esc",
			state:    conflictState,
			msg:      tea.KeyMsg{Type: tea.KeyEsc},
			expState: menuState,
		},
		{
			name:  "нажатие n",
			state: listState,
			setupModel: func(m *AppModel) {
				m.list.sType = models.SecretType("card")
			},
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}},
			expState: formState,
		},
		{
			name:     "возврат из formState в listState по Esc",
			state:    formState,
			msg:      tea.KeyMsg{Type: tea.KeyEsc},
			expState: listState,
		},
		{
			name:  "успешное сохранение формы",
			state: formState,
			setupModel: func(m *AppModel) {
				m.form.saved = true
			},
			msg:      tea.KeyMsg{Type: tea.KeyEnter},
			expState: listState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			service.On("GetAll", mock.Anything, mock.Anything).Return([]models.Secret{}, nil).Maybe()
			if tt.setupMock != nil {
				tt.setupMock(service)
			}
			m := NewAppModel(service, syncChan, occurrenceChan, Bulder{Version: "N/A", Date: "N/A", Commit: "N/A"})
			m.state = tt.state
			if tt.setupModel != nil {
				tt.setupModel(&m)
			}
			msg := tt.msg
			if tt.wantQuit {
				msg = tea.KeyMsg{Type: tea.KeyCtrlC, Runes: []rune{}}
			}
			updatedModel, cmd := m.Update(msg)
			model := updatedModel.(AppModel)
			assert.Equal(t, tt.expState, model.state)
			if tt.wantQuit {
				if assert.NotNil(t, cmd) {
					assert.IsType(t, tea.QuitMsg{}, cmd())
				}
			}
		})
	}
}

func TestAppModel_SyncAndInit(t *testing.T) {
	syncChan := make(chan struct{}, 1)
	occurrenceChan := make(chan struct{}, 1)
	service := mocks.NewSecretService(t)
	service.On("GetAll", mock.Anything, mock.Anything).Return([]models.Secret{}, nil).Maybe()
	m := NewAppModel(service, syncChan, occurrenceChan, Bulder{Version: "N/A", Date: "N/A", Commit: "N/A"})
	cmd := m.Init()
	assert.NotNil(t, cmd)
	occurrenceChan <- struct{}{}
	syncCmd := Sync(occurrenceChan)
	if assert.NotNil(t, syncCmd) {
		msg := syncCmd()
		assert.IsType(t, SyncOccurrenceMsg{}, msg)
	}
	close(occurrenceChan)
	syncClosed := Sync(occurrenceChan)
	if assert.NotNil(t, syncClosed) {
		assert.Nil(t, syncClosed())
	}
	nilSync := Sync(nil)
	if assert.NotNil(t, nilSync) {
		assert.Nil(t, nilSync())
	}
}

func TestAppModel_View(t *testing.T) {
	service := mocks.NewSecretService(t)
	service.On("GetAll", mock.Anything, mock.Anything).Return([]models.Secret{}, nil).Maybe()
	m := NewAppModel(service, nil, nil, Bulder{Version: "N/A", Date: "N/A", Commit: "N/A"})
	tests := []struct {
		name  string
		state sessionState
	}{
		{
			name:  "отображение authState",
			state: authState,
		},
		{
			name:  "отображение authFormState",
			state: authFormState,
		},
		{
			name:  "отображение menuState",
			state: menuState,
		},
		{
			name:  "отображение listState",
			state: listState,
		},
		{
			name:  "отображение formState",
			state: formState,
		},
		{
			name:  "отображение conflictState",
			state: conflictState,
		},
		{
			name:  "Дефолтное состояние ошибки",
			state: sessionState(10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.state = tt.state
			viewStr := m.View()
			if tt.state == sessionState(10) {
				assert.NotEmpty(t, viewStr)
			} else {
				assert.NotNil(t, viewStr)
			}
		})
	}
}
