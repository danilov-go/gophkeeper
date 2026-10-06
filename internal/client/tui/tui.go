package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type SecretService interface {
	Save(ctx context.Context, meta models.MetaData, data models.SecretData) (int, error)
	GetAll(ctx context.Context, sType models.SecretType) ([]models.Secret, error)
	Update(ctx context.Context, id int, meta models.MetaData, data models.SecretData) error
	Delete(ctx context.Context, id int) error
	Get(ctx context.Context, id int) (models.Secret, error)
	GetFile(path string) ([]byte, error)
	ExportFile(path string, data []byte) error
}

type sessionState int

const (
	menuState = iota
	listState
	formState
)

type AppModel struct {
	state   sessionState
	service SecretService
	menu    MenuModel
	list    ListModel
	form    FormModel
}

func NewAppModel(storage SecretService) AppModel {
	return AppModel{
		state:   menuState,
		service: storage,
		menu:    NewMenuModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	switch m.state {
	case menuState:
		updatedMenu, menuCmd := m.menu.Update(msg)
		m.menu = updatedMenu.(MenuModel)
		if m.menu.chosen {
			m.menu.chosen = false
			m.state = listState
			m.list = NewListModel(m.service, m.menu.options[m.menu.cursor].sType)
			return m, m.list.Init()
		}
		return m, menuCmd
	case listState:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "n", "+":
				m.form = NewFormModel(m.service, m.list.sType)
				m.state = formState
				return m, m.form.Init()
			case "enter":
				if len(m.list.secrets) == 0 {
					return m, nil
				}
				secret := m.list.secrets[m.list.cursor]
				m.form = NewEditForm(m.service, secret)
				m.state = formState
				return m, m.form.Init()
			case "esc", "backspace":
				m.state = menuState
				return m, nil
			}
		}
		updatedList, listCmd := m.list.Update(msg)
		m.list = updatedList.(ListModel)
		return m, listCmd
	case formState:
		updatedForm, formCmd := m.form.Update(msg)
		m.form = updatedForm.(FormModel)
		if m.form.saved {
			m.form.saved = false
			m.state = listState
			return m, m.list.Init()
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.form.saved = false
			m.state = listState
			return m, nil
		}
		return m, formCmd
	}
	return m, nil
}

func (m AppModel) View() string {
	switch m.state {
	case menuState:
		return m.menu.View()
	case listState:
		return m.list.View()
	case formState:
		return m.form.View()
	default:
		return "Неизвестное состояние приложения."
	}
}
