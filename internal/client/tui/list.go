package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/pquerna/otp/totp"
)

// MsgList событие для вывода текстовых сообщений на экран.
type MsgList struct {
	Text string
}

// MsgOTP событие таймера для обновления OTP-кодов.
type MsgOTP time.Time

// ListModel управляет экраном просмотра, поиска и удаления секретов.
type ListModel struct {
	service   SecretService
	sType     models.SecretType
	secrets   []models.Secret
	cursor    int
	err       error
	statusMsg string
	tikerOTP  int
	search    textinput.Model
	searching bool
	help      help.Model
	delete    bool
}

// NewListModel создает новый экземпляр ListModel.
func NewListModel(srv SecretService, sType models.SecretType) ListModel {
	ti := textinput.New()
	ti.Placeholder = "Введите для фильтрации по названию..."
	ti.CharLimit = 40
	return ListModel{
		service:   srv,
		sType:     sType,
		secrets:   []models.Secret{},
		tikerOTP:  int(30 - (time.Now().Unix() % 30)),
		search:    ti,
		searching: false,
		help:      help.New(),
	}
}

// Init загружает список секретов и запускает таймер OTP кодов.
func (m ListModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			data, err := m.service.GetAll(ctx, m.sType)
			if err != nil {
				return err
			}
			return data
		},
		Tick(),
	)
}

// Tick запускает секундный таймер для обновления OTP кодов.
func Tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return MsgOTP(t)
	})
}

// Update управляет логикой списка
func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.searching {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "enter", "esc":
				m.searching = false
				m.search.Blur()
				return m, nil
			}
		}
		m.search, cmd = m.search.Update(msg)
		m.cursor = 0
		return m, cmd
	}
	switch msg := msg.(type) {
	case MsgOTP:
		m.tikerOTP = int(30 - (time.Now().Unix() % 30))
		return m, Tick()
	case SyncOccurrenceMsg:
		if m.delete {
			return m, nil
		}
		m.statusMsg = "Данные обновлены"
		return m, m.Init()
	case tea.KeyMsg:
		secrets := m.filtered()
		switch msg.String() {
		case "/":
			m.searching = true
			m.statusMsg = ""
			return m, m.search.Focus()
		case "up":
			m.statusMsg = ""
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			m.statusMsg = ""
			if m.cursor < len(secrets)-1 {
				m.cursor++
			}
		case "d", "в":
			m.statusMsg = ""
			if len(secrets) == 0 {
				return m, nil
			}
			secret := secrets[m.cursor]
			m.delete = true
			m.secrets = slices.DeleteFunc(m.secrets, func(s models.Secret) bool {
				return s.ID == secret.ID
			})
			if m.cursor > 0 && m.cursor == len(secrets)-1 {
				m.cursor--
			}
			return m, m.deleteSecret(secret.ID)
		case "s", "ы":
			if len(secrets) == 0 || m.sType != models.TypeBinary {
				return m, nil
			}
			secret := secrets[m.cursor]
			return m, m.exportCmd(secret)
		}
	case []models.Secret:
		m.secrets = msg
		m.err = nil
		secrets := m.filtered()
		if m.cursor >= len(secrets) && len(secrets) > 0 {
			m.cursor = len(secrets) - 1
		}
	case error:
		m.delete = false
		m.err = msg
	case MsgList:
		m.statusMsg = msg.Text
		m.err = nil
		m.delete = false
		return m, nil
	}
	return m, nil
}

func (m ListModel) deleteSecret(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := m.service.Delete(ctx, id)
		if err != nil {
			return err
		}
		time.Sleep(time.Millisecond * 600)
		return MsgList{Text: "Запись успешно удалена"}
	}
}

// View отрисовывает список секретов, форматируя строки под каждый тип данных.
func (m ListModel) View() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Категория: %s \n\n", m.sType))
	if m.searching || m.search.Value() != "" {
		s.WriteString("Поиск: " + m.search.View() + "\n\n")
	}
	if m.statusMsg != "" {
		s.WriteString(fmt.Sprintf("%s\n\n", m.statusMsg))
	}
	secrets := m.filtered()
	if m.err != nil {
		errStr := m.err.Error()
		if strings.Contains(errStr, "пустое") || strings.Contains(errStr, "отсутствуют") {
			s.WriteString("Список пуст.\n\n")
		} else {
			s.WriteString(fmt.Sprintf("Ошибка: %v\n\n", m.err))
		}
	} else if len(secrets) == 0 {
		s.WriteString("Список пуст.\n\n")
	} else {
		for i, secret := range secrets {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			switch v := secret.Meta.(type) {
			case *models.MetaCard:
				s.WriteString(fmt.Sprintf("%s Карта: %s [%s]\n", cursor, v.Name, v.BankName))
			case *models.MetaLoginPassword:
				otpCode := "Не настроен"
				if lp, ok := secret.Data.(*models.LoginPassword); ok && lp.SecretTOTP != "" {
					var err error
					otpCode, err = totp.GenerateCode(lp.SecretTOTP, time.Now())
					if err != nil {
						otpCode = "Невалидный ключ"
					}
				}
				s.WriteString(fmt.Sprintf("%s Аккаунт: %s (%s) | OTP: %s, Время действия OTP: %d\n", cursor, v.Name, v.URL, otpCode, m.tikerOTP))
			case *models.MetaText:
				runes := []rune(v.Name)
				preview := string(runes)
				if len(runes) > 30 {
					preview = string(runes[:27]) + "..."
				}
				s.WriteString(fmt.Sprintf("%s Заметка: %s\n", cursor, preview))
			case *models.MetaBinary:
				s.WriteString(fmt.Sprintf("%s Файл: %s (%s) | %d байт\n", cursor, v.Name, v.Type, v.Size))
			default:
				s.WriteString(fmt.Sprintf("%s Данные неизвестного типа\n", cursor))
			}
		}
	}
	listKeys := keys
	if m.sType == models.TypeBinary {
		listKeys.mode = modeBinary
	} else {
		listKeys.mode = modeList
	}
	s.WriteString("\n" + m.help.View(listKeys) + "\n")
	return s.String()
}

func (m ListModel) filtered() []models.Secret {
	query := strings.ToLower(m.search.Value())
	if query == "" {
		return m.secrets
	}
	var secrets []models.Secret
	for _, secret := range m.secrets {
		name := ""
		switch v := secret.Meta.(type) {
		case *models.MetaCard:
			name = v.Name
		case *models.MetaLoginPassword:
			name = v.Name
		case *models.MetaText:
			name = v.Name
		case *models.MetaBinary:
			name = v.Name
		}
		if strings.Contains(strings.ToLower(name), query) {
			secrets = append(secrets, secret)
		}
	}
	return secrets
}

func (m ListModel) exportCmd(secret models.Secret) tea.Cmd {
	return func() tea.Msg {
		meta, okMeta := secret.Meta.(*models.MetaBinary)
		data, okData := secret.Data.(*models.BinaryData)
		if !okMeta || !okData {
			return errors.New("ошибка: данные файла повреждены или пустые")
		}
		fileName := meta.Name
		if !strings.HasSuffix(strings.ToLower(fileName), strings.ToLower(meta.Type)) {
			fileName = fileName + meta.Type
		}
		err := m.service.ExportFile(fileName, data.Data)
		if err != nil {
			return err
		}
		return MsgList{
			Text: "Файл успешно сохранен",
		}
	}
}
