package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
)

// ConflictModel управляет экраном просмотра и разрешения конфликтов синхронизации.
type ConflictModel struct {
	service   SecretService
	conflicts []models.Conflict
	cursor    int
	err       error
	help      help.Model
}

// NewConflictModel создает новый экземпляр ConflictModel.
func NewConflictModel(srv SecretService) ConflictModel {
	return ConflictModel{
		service: srv,
		help:    help.New(),
	}
}

// Init запускает загрузку списка конфликтов синхронизации.
func (m ConflictModel) Init() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		list, err := m.service.GetConflicts(ctx)
		if err != nil {
			return err
		}
		return list
	}
}

// Update обрабатывает выбор конфликта и применение серверной или локальной версии.
func (m ConflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []models.Conflict:
		m.conflicts = msg
		m.cursor = 0
		return m, nil
	case error:
		m.err = msg
		return m, nil
	case string:
		if msg == "conflict_resolved" {
			return m, m.Init()
		}
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
			conflict := m.conflicts[m.cursor]
			return m, m.fixConflict(conflict.ID, true)

		case "2":
			conflict := m.conflicts[m.cursor]
			return m, m.fixConflict(conflict.ID, false)
		}
	}
	return m, nil
}

func (m ConflictModel) fixConflict(id string, keepServer bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := m.service.FixConflict(ctx, id, keepServer)
		if err != nil {
			return err
		}
		return "conflict_resolved"
	}
}

// View отрисовывает список конфликтов.
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
		_, _ = fmt.Fprintf(&b, "%s Конфликт в секрете ID %s (Тип: %s)\n", cursorStr, c.ID, c.Local.Type)
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
