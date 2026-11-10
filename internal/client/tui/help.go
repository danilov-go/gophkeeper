package tui

import "github.com/charmbracelet/bubbles/key"

type helpMode int

const (
	modeMenu helpMode = iota
	modeList
	modeConflict
	modeForm
	modeBinary
)

type keyMap struct {
	mode       helpMode
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Add        key.Binding
	Search     key.Binding
	Delete     key.Binding
	Export     key.Binding
	Logout     key.Binding
	Back       key.Binding
	Conflict   key.Binding
	KeepServer key.Binding
	KeepLocal  key.Binding
	Hide       key.Binding
}

// MenuHelp возвращает информацию о клавишах управления для главного меню.
func (k keyMap) MenuHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Logout}
}

// ListHelp возвращает информацию о клавишах управления для экрана со списками.
func (k keyMap) ListHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Add, k.Enter, k.Delete, k.Search, k.Back}
}

// BinaryHelp возвращает информацию о клавишах управления для экрана с бинарными данными.
func (k keyMap) BinaryHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Add, k.Enter, k.Delete, k.Search, k.Back, k.Export}
}

// BinaryHelp возвращает информацию о клавишах управления для экрана редактирования записи.
func (k keyMap) FormHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Hide, k.Back}
}

// ConflictHelp возвращает информацию о клавишах управления для экрана разрешения конфликтов.
func (k keyMap) ConflictHelp() []key.Binding {
	return []key.Binding{k.KeepServer, k.KeepLocal, k.Back}
}

// ShortHelp возвращает информацию о клавишах управления для текущего режима экрана.
func (k keyMap) ShortHelp() []key.Binding {
	switch k.mode {
	case modeList:
		return k.ListHelp()
	case modeConflict:
		return k.ConflictHelp()
	case modeForm:
		return k.FormHelp()
	case modeBinary:
		return k.BinaryHelp()
	}
	return k.MenuHelp()
}

// FullHelp возвращает полную информацию о клавишах управления для интерфейса.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ListHelp()}
}

var keys = keyMap{
	mode:       modeMenu,
	Up:         key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "вверх")),
	Down:       key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "вниз")),
	Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "выбрать")),
	Add:        key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "создать")),
	Search:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "поиск")),
	Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "удалить")),
	Export:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "экспорт")),
	Logout:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "выйти")),
	Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "назад")),
	Conflict:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "просмотреть конфликты")),
	KeepServer: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "сохранить версию сервера")),
	KeepLocal:  key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "сохранить локальную версию")),
	Hide:       key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r ", "показать ")),
}
