package tui

import "github.com/charmbracelet/bubbles/key"

type helpMode int

const (
	modeMenu helpMode = iota
	modeList
	modeConflict
)

type keyMap struct {
	mode       helpMode
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Search     key.Binding
	Delete     key.Binding
	Export     key.Binding
	Logout     key.Binding
	Back       key.Binding
	Conflict   key.Binding
	KeepServer key.Binding
	KeepLocal  key.Binding
}

func (k keyMap) MenuHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Logout}
}

func (k keyMap) ListHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Search, k.Delete, k.Export, k.Back}
}
func (k keyMap) ConflictHelp() []key.Binding {
	return []key.Binding{k.KeepServer, k.KeepLocal, k.Back}
}

func (k keyMap) ShortHelp() []key.Binding {
	switch k.mode {
	case modeList:
		return k.ListHelp()
	case modeConflict:
		return k.ConflictHelp()
	}
	return k.MenuHelp()
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ListHelp()}
}

var keys = keyMap{
	mode:       modeMenu,
	Up:         key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "вверх")),
	Down:       key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "вниз")),
	Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "выбрать")),
	Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "поиск")),
	Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "удалить")),
	Export:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "экспорт")),
	Logout:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "выйти")),
	Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "назад")),
	Conflict:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "просмотреть конфликты")),
	KeepServer: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "сохранить версию сервера")),
	KeepLocal:  key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "сохранить локальную версию")),
}
