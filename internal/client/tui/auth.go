package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// AuthMenuModel управляет стартовым экраном выбора между входом и регистрацией.
type AuthMenuModel struct {
	cursor       int
	options      []string
	registration bool
	chosen       bool
	bulder       Bulder
}

// NewAuthMenuModel создает новый экземпляр AuthMenuModel.
func NewAuthMenuModel(bulder Bulder) AuthMenuModel {
	return AuthMenuModel{
		options: []string{"Войти в аккаунт", "Зарегистрироваться"},
		bulder:  bulder,
	}
}

func (m AuthMenuModel) Init() tea.Cmd {
	return nil
}

// Update обрабатывает выбор между входом и регистрацией.
func (m AuthMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.registration = m.cursor == 1
			m.chosen = true
			return m, nil
		}
	}
	return m, nil
}

// View отрисовывает пункты выбора авторизации и информацию о версии сборки.
func (m AuthMenuModel) View() string {
	var s strings.Builder
	for i, option := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, option))
	}
	s.WriteString("\n" + "Версия: " + m.bulder.Version + " " + "Дата: " + m.bulder.Date)
	return s.String()
}
