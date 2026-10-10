package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type AuthMenuModel struct {
	cursor         int
	options        []string
	isRegistration bool
	chosen         bool
}

func NewAuthMenuModel() AuthMenuModel {
	return AuthMenuModel{
		options: []string{"Войти в аккаунт", "Зарегистрироваться"},
	}
}

func (m AuthMenuModel) Init() tea.Cmd {
	return nil
}

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
			m.isRegistration = m.cursor == 1
			m.chosen = true
			return m, nil
		}
	}
	return m, nil
}

func (m AuthMenuModel) View() string {
	var s strings.Builder
	for i, option := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, option))
	}
	return s.String()
}
