package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type MenuOption struct {
	title string
	sType models.SecretType
}

type MenuModel struct {
	options []MenuOption
	cursor  int
	chosen  bool
}

func NewMenuModel() MenuModel {
	return MenuModel{
		options: []MenuOption{
			{title: "Банковские карты", sType: models.TypeCard},
			{title: "Пароли и логины", sType: models.TypeLoginPassword},
			{title: "Текстовые данные", sType: models.TypeText},
			{title: "Бинарники", sType: models.TypeBinary},
		},
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.chosen = true
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	var s strings.Builder
	s.WriteString("Главное меню\n\n")
	s.WriteString("Выберите категорию секретов для просмотра:\n\n")
	for i, option := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s.WriteString(cursor + " " + option.title + "\n")
	}
	return s.String()
}
