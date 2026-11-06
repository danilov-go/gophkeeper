package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/danilov-go/gophkeeper/internal/models"
)

func TestMenuModel_Init(t *testing.T) {
	m := NewMenuModel()
	assert.Nil(t, m.Init())
}

func TestMenuModel_NewMenuModel(t *testing.T) {
	m := NewMenuModel()
	assert.Len(t, m.options, 4)
	assert.Equal(t, 0, m.cursor)
	assert.False(t, m.chosen)
	options := []MenuOption{
		{title: "Банковские карты", sType: models.TypeCard},
		{title: "Пароли и логины", sType: models.TypeLoginPassword},
		{title: "Текстовые данные", sType: models.TypeText},
		{title: "Бинарники", sType: models.TypeBinary},
	}
	assert.Equal(t, options, m.options)
}

func TestMenuModel_Update(t *testing.T) {
	tests := []struct {
		name          string
		initialCursor int
		msg           tea.Msg
		cursor        int
		wantEnter     bool
	}{
		{
			name:          "нажатие down",
			initialCursor: 0,
			msg:           tea.KeyMsg{Type: tea.KeyDown},
			cursor:        1,
			wantEnter:     false,
		},
		{
			name:          "нажатие up",
			initialCursor: 2,
			msg:           tea.KeyMsg{Type: tea.KeyUp},
			cursor:        1,
			wantEnter:     false,
		},
		{
			name:          "нажатие enter",
			initialCursor: 1,
			msg:           tea.KeyMsg{Type: tea.KeyEnter},
			cursor:        1,
			wantEnter:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMenuModel()
			m.cursor = tt.initialCursor
			currentMsg := tt.msg
			if key, ok := tt.msg.(tea.KeyMsg); ok {
				if string(key.Runes) == "up" || key.Type == tea.KeyUp {
					currentMsg = tea.KeyMsg{Type: tea.KeyUp}
				} else if string(key.Runes) == "down" || key.Type == tea.KeyDown {
					currentMsg = tea.KeyMsg{Type: tea.KeyDown}
				}
			}
			updatedModel, cmd := m.Update(currentMsg)
			model := updatedModel.(MenuModel)
			assert.Nil(t, cmd)
			assert.Equal(t, tt.cursor, model.cursor)
			assert.Equal(t, tt.wantEnter, model.chosen)
		})
	}
}

func TestMenuModel_View(t *testing.T) {
	tests := []struct {
		name    string
		cursor  int
		expText []string
	}{
		{
			name:   "меню с курсором на первой строке",
			cursor: 0,
			expText: []string{
				"Главное меню",
				"Выберите категорию секретов для просмотра:",
				"> Банковские карты",
				"  Пароли и логины",
			},
		},
		{
			name:   "меню с курсором на текстовых данных",
			cursor: 2,
			expText: []string{
				"  Банковские карты",
				"> Текстовые данные",
				"  Бинарники",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMenuModel()
			m.cursor = tt.cursor
			text := m.View()
			for _, expText := range tt.expText {
				assert.Contains(t, text, expText)
			}
		})
	}
}
