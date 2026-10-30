package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type ConflictModel struct {
	service   SecretService
	conflicts []models.Conflict
	cursor    int
	err       error
	help      help.Model
}

func NewConflictModel(srv SecretService) ConflictModel {
	return ConflictModel{
		service: srv,
		help:    help.New(),
	}
}

func (m ConflictModel) Init() tea.Cmd {
	return func() tea.Msg {
		list, err := m.service.GetConflicts(context.Background())
		if err != nil {
			return err
		}
		return list
	}
}

func (m ConflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []models.Conflict:
		m.conflicts = msg
		m.cursor = 0
		return m, nil
	case error:
		m.err = msg
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, nil
		}
		if len(m.conflicts) == 0 {
			return m, nil
		}

		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.conflicts)-1 {
				m.cursor++
			}
		case "1":
			current := m.conflicts[m.cursor]
			_ = m.service.FixConflict(context.Background(), current.ID, true)
			return m, m.Init()
		case "2":
			current := m.conflicts[m.cursor]
			_ = m.service.FixConflict(context.Background(), current.ID, false)
			return m, m.Init()
		}
	}
	return m, nil
}

func (m ConflictModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Ошибка загрузки конфликтов: %v\n,", m.err)
	}
	if len(m.conflicts) == 0 {
		return "Отложенных конфликтов синхронизации не обнаружено"
	}
	var b strings.Builder
	b.WriteString("Разрешение конфликтов синхронизации\n")
	for i, c := range m.conflicts {
		cursorStr := " "
		if i == m.cursor {
			cursorStr = ">"
		}
		b.WriteString(fmt.Sprintf("%s Конфликт в секрете ID %s (Тип: %s)\n", cursorStr, c.ID, c.Local.Type))
		if i == m.cursor {
			serverForm := NewEditForm(m.service, c.Server)
			for idx := range serverForm.inputs {
				serverForm.inputs[idx].Blur()
			}
			localForm := NewEditForm(m.service, c.Local)
			for idx := range localForm.inputs {
				localForm.inputs[idx].Blur()
			}
			b.WriteString(" Версия сервера \n")
			b.WriteString(serverForm.View() + "\n")
			b.WriteString(" Локальная версия\n")
			b.WriteString(localForm.View() + "\n")
		}
	}
	conflictKeys := keys
	conflictKeys.mode = modeConflict
	b.WriteString("\n" + m.help.View(conflictKeys) + "\n")
	return b.String()
}
