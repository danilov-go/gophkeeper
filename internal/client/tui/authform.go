package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type AuthFormModel struct {
	service      SecretService
	inputs       []textinput.Model
	focus        int
	success      bool
	err          error
	registration bool
}

func NewAuthFormModel(srv SecretService, registration bool) AuthFormModel {
	inputs := make([]textinput.Model, 2)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Введите логин..."
	inputs[0].CharLimit = 40
	inputs[0].Focus()
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Введите пароль..."
	inputs[1].CharLimit = 64
	inputs[1].EchoMode = textinput.EchoPassword

	return AuthFormModel{
		service:      srv,
		registration: registration,
		inputs:       inputs,
	}
}

func (m AuthFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AuthFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down", "enter":
			if m.focus == len(m.inputs)-1 {
				return m, m.submitCmd()
			}
			m.focus++
			return m.switchFocus()
		case "up":
			if m.focus > 0 {
				m.focus--
			}
			return m.switchFocus()
		}
	case string:
		if msg == "auth_success" {
			m.success = true
		}
	case error:
		m.err = msg
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m AuthFormModel) View() string {
	var s strings.Builder
	if m.registration {
		s.WriteString("Регистрация\n\n")
	} else {
		s.WriteString("Авторизация\n\n")
	}
	if m.err != nil {
		s.WriteString(fmt.Sprintf("Ошибка: %v\n\n", m.err))
	}
	if m.success {
		s.WriteString("Успешно!\n\n")
		return s.String()
	}
	labels := []string{"Логин ", "Пароль"}
	for i := range m.inputs {
		s.WriteString(fmt.Sprintf(" %s: %s\n", labels[i], m.inputs[i].View()))
	}
	return s.String()
}

func (m AuthFormModel) switchFocus() (AuthFormModel, tea.Cmd) {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i < len(m.inputs); i++ {
		if i == m.focus {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *AuthFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i < len(m.inputs); i++ {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m AuthFormModel) submitCmd() tea.Cmd {
	return func() tea.Msg {
		login := m.inputs[0].Value()
		password := m.inputs[1].Value()

		if login == "" || password == "" {
			return errors.New("поля не могут быть пустыми")
		}
		var err error
		if m.registration {
			err = m.service.Register(context.Background(), login, password)
		} else {
			err = m.service.Login(context.Background(), login, password)
		}
		if err != nil {
			return err
		}
		return "auth_success"
	}
}
