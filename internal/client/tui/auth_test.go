package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestAuthMenuModel_Init(t *testing.T) {
	m := NewAuthMenuModel(Bulder{Version: "N/A", Date: "N/A", Commit: "N/A"})
	assert.Nil(t, m.Init())
}

func TestAuthMenuModel_Update(t *testing.T) {
	type testCase struct {
		name             string
		initCursor       int
		msg              tea.Msg
		cursor           int
		wantRegistration bool
		wantEnter        bool
	}

	tests := []testCase{
		{
			name:             "нажатие down",
			initCursor:       0,
			msg:              tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("down")},
			cursor:           1,
			wantRegistration: false,
			wantEnter:        false,
		},

		{
			name:             "нажатие up",
			initCursor:       1,
			msg:              tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("up")},
			cursor:           0,
			wantRegistration: false,
			wantEnter:        false,
		},
		{
			name:             "нажатие enter (выбор авторизации)",
			initCursor:       0,
			msg:              tea.KeyMsg{Type: tea.KeyEnter},
			cursor:           0,
			wantRegistration: false,
			wantEnter:        true,
		},
		{
			name:             "нажатие enter (выбор регистрации)",
			initCursor:       1,
			msg:              tea.KeyMsg{Type: tea.KeyEnter},
			cursor:           1,
			wantRegistration: true,
			wantEnter:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewAuthMenuModel(Bulder{Version: "N/A", Date: "N/A", Commit: "N/A"})
			m.cursor = tt.initCursor
			currentMsg := tt.msg
			if key, ok := tt.msg.(tea.KeyMsg); ok && (string(key.Runes) == "up" || string(key.Runes) == "down") {
				if string(key.Runes) == "up" {
					currentMsg = tea.KeyMsg{Type: tea.KeyUp}
				} else {
					currentMsg = tea.KeyMsg{Type: tea.KeyDown}
				}
			}
			updatedModel, cmd := m.Update(currentMsg)
			model := updatedModel.(AuthMenuModel)
			assert.Nil(t, cmd)
			assert.Equal(t, tt.cursor, model.cursor)
			assert.Equal(t, tt.wantRegistration, model.isRegistration)
			assert.Equal(t, tt.wantEnter, model.chosen)
		})
	}
}

func TestAuthMenuModel_View(t *testing.T) {
	tests := []struct {
		name   string
		cursor int
		text   []string
	}{
		{
			name:   "курсор на первом пункте меню",
			cursor: 0,
			text: []string{
				"> Войти в аккаунт",
				"  Зарегистрироваться",
			},
		},
		{
			name:   "курсор на втором пункте меню",
			cursor: 1,
			text: []string{
				"  Войти в аккаунт",
				"> Зарегистрироваться",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewAuthMenuModel(Bulder{Version: "N/A", Date: "N/A", Commit: "N/A"})
			m.cursor = tt.cursor
			text := m.View()
			for _, expText := range tt.text {
				assert.Contains(t, text, expText)
			}
		})
	}
}
