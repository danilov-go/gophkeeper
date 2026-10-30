package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type SyncOccurrenceMsg struct{}

//go:generate mockery --name=SecretService --case=underscore --output=./mocks --with-expecter
type SecretService interface {
	Register(ctx context.Context, login, password string) error
	Login(ctx context.Context, login, password string) error
	Save(ctx context.Context, meta models.MetaData, data models.SecretData) (string, error)
	GetAll(ctx context.Context, sType models.SecretType) ([]models.Secret, error)
	Update(ctx context.Context, id string, meta models.MetaData, data models.SecretData) error
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (models.Secret, error)
	GetFile(path string) ([]byte, error)
	ExportFile(path string, data []byte) error
	Logout()
	GetConflicts(ctx context.Context) ([]models.Conflict, error)
	FixConflict(ctx context.Context, id string, keepServer bool) error
}

type sessionState int

const (
	authState sessionState = iota
	authFormState
	menuState
	listState
	formState
	conflictState
)

type AppModel struct {
	state          sessionState
	service        SecretService
	syncChan       chan<- struct{}
	occurrenceChan <-chan struct{}
	authMenu       AuthMenuModel
	authForm       AuthFormModel
	menu           MenuModel
	list           ListModel
	form           FormModel
	conflict       ConflictModel
	bulder         Bulder
}

type Bulder struct {
	Version string
	Date    string
	Commit  string
}

func NewAppModel(storage SecretService, syncChan chan<- struct{}, occurrenceChan <-chan struct{}, bulder Bulder) AppModel {
	return AppModel{
		state:          authState,
		service:        storage,
		syncChan:       syncChan,
		occurrenceChan: occurrenceChan,
		authMenu:       NewAuthMenuModel(bulder),
		list:           NewListModel(storage, models.SecretType("")),
		conflict:       NewConflictModel(storage),
		bulder:         bulder,
	}
}

func Sync(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		_, ok := <-ch
		if !ok {
			return nil
		}
		return SyncOccurrenceMsg{}
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.authMenu.Init(),
		Sync(m.occurrenceChan),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	switch msg.(type) {
	case SyncOccurrenceMsg:
		if m.state == listState && m.list.sType != "" {
			updatedList, listCmd := m.list.Update(SyncOccurrenceMsg{})
			m.list = updatedList.(ListModel)
			return m, tea.Batch(listCmd, Sync(m.occurrenceChan))
		}
		return m, Sync(m.occurrenceChan)
	}
	switch m.state {
	case authState:
		updatedAuthMenu, authMenuCmd := m.authMenu.Update(msg)
		m.authMenu = updatedAuthMenu.(AuthMenuModel)
		if m.authMenu.chosen {
			m.authMenu.chosen = false
			m.state = authFormState
			m.authForm = NewAuthFormModel(m.service, m.authMenu.isRegistration)
			return m, m.authForm.Init()
		}
		return m, authMenuCmd
	case authFormState:
		updatedAuthForm, authFormCmd := m.authForm.Update(msg)
		m.authForm = updatedAuthForm.(AuthFormModel)
		if m.authForm.success {
			m.authForm.success = false
			m.state = menuState
			m.menu = NewMenuModel()
			if m.syncChan != nil {
				select {
				case m.syncChan <- struct{}{}:
				default:
				}
			}
			return m, m.menu.Init()
		}
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.state = authState
			return m, nil
		}
		return m, authFormCmd
	case menuState:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "q" {
				m.service.Logout()
				m.state = authState
				m.authMenu = NewAuthMenuModel(m.bulder)
				return m, m.authMenu.Init()
			}
			if keyMsg.String() == "c" {
				m.state = conflictState
				m.conflict = NewConflictModel(m.service)
				return m, m.conflict.Init()
			}
		}
		updatedMenu, menuCmd := m.menu.Update(msg)
		m.menu = updatedMenu.(MenuModel)
		if m.menu.chosen {
			m.menu.chosen = false
			m.state = listState
			selectedType := m.menu.options[m.menu.cursor].sType
			m.list = NewListModel(m.service, selectedType)
			if cachedData, err := m.service.GetAll(context.Background(), selectedType); err == nil {
				m.list.secrets = cachedData
			}
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
	case conflictState:
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			m.state = menuState
			return m, nil
		}
		updatedConflict, conflictCmd := m.conflict.Update(msg)
		m.conflict = updatedConflict.(ConflictModel)
		return m, conflictCmd
	}
	return m, nil
}

func (m AppModel) View() string {
	switch m.state {
	case authState:
		return m.authMenu.View()
	case authFormState:
		return m.authForm.View()
	case menuState:
		return m.menu.View()
	case listState:
		return m.list.View()
	case formState:
		return m.form.View()
	case conflictState:
		return m.conflict.View()
	default:
		return "Неизвестное состояние приложения."
	}
}
