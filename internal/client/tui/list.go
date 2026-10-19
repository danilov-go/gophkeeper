package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/pquerna/otp/totp"
)

type MsgList struct {
	Text string
}

type MsgOTP time.Time

type ListModel struct {
	service   SecretService
	sType     models.SecretType
	secrets   []models.Secret
	cursor    int
	err       error
	statusMsg string
	tikerTOTP int
}

func NewListModel(srv SecretService, sType models.SecretType) ListModel {
	return ListModel{
		service:   srv,
		sType:     sType,
		secrets:   []models.Secret{},
		tikerTOTP: int(30 - (time.Now().Unix() % 30)),
	}
}

func (m ListModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			data, err := m.service.GetAll(context.Background(), m.sType)
			if err != nil {
				return err
			}
			return data
		},
		Tick(),
	)
}

func Tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return MsgOTP(t)
	})
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case MsgOTP:
		m.tikerTOTP = int(30 - (time.Now().Unix() % 30))
		return m, Tick()
	case SyncOccurrenceMsg:
		m.statusMsg = "Данные обновлены"
		return m, m.Init()
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.statusMsg = ""
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			m.statusMsg = ""
			if m.cursor < len(m.secrets)-1 {
				m.cursor++
			}
		case "d":
			m.statusMsg = ""
			if len(m.secrets) == 0 {
				return m, nil
			}
			targetSecret := m.secrets[m.cursor]
			err := m.service.Delete(context.Background(), targetSecret.ID)
			if err != nil {
				m.err = err
				return m, nil
			}
			if m.cursor > 0 && m.cursor == len(m.secrets)-1 {
				m.cursor--
			}
			return m, m.Init()
		case "s":
			if len(m.secrets) == 0 || m.sType != models.TypeBinary {
				return m, nil
			}
			targetSecret := m.secrets[m.cursor]
			return m, m.exportCmd(targetSecret)
		}
	case []models.Secret:
		m.secrets = msg
		m.err = nil
		if m.cursor >= len(m.secrets) && len(m.secrets) > 0 {
			m.cursor = len(m.secrets) - 1
		}
	case error:
		m.err = msg
	case MsgList:
		m.statusMsg = msg.Text
		m.err = nil
	}
	return m, nil
}

func (m ListModel) View() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Категория: %s \n\n", m.sType))
	if m.statusMsg != "" {
		s.WriteString(fmt.Sprintf("%s\n\n", m.statusMsg))
	}
	if m.err != nil {
		errStr := m.err.Error()
		if strings.Contains(errStr, "пустое") || strings.Contains(errStr, "отсутствуют") {
			s.WriteString("Список пуст.\n\n")
		} else {
			s.WriteString(fmt.Sprintf("Ошибка: %v\n\n", m.err))
		}
	} else if len(m.secrets) == 0 {
		s.WriteString("Список пуст.\n\n")
	} else {
		for i, secret := range m.secrets {
			cursor := " "
			if m.cursor == i {
				cursor = ">"
			}
			switch v := secret.Meta.(type) {
			case *models.MetaCard:
				s.WriteString(fmt.Sprintf("%s [ID: %s] Карта: %s [%s]\n", cursor, secret.ID, v.Name, v.BankName))

			case *models.MetaLoginPassword:
				otpCode := "Not configured"
				if lp, ok := secret.Data.(*models.LoginPassword); ok && lp.SecretTOTP != "" {
					var err error
					otpCode, err = totp.GenerateCode(lp.SecretTOTP, time.Now())
					if err != nil {
						otpCode = "Invalid Secret"
					}
				}
				s.WriteString(fmt.Sprintf("%s [ID: %s] Аккаунт: %s (%s) | 2FA: %s, Время действия TOTP: %d\n", cursor, secret.ID, v.Name, v.URL, otpCode, m.tikerTOTP))

			case *models.MetaText:
				preview := v.Name
				if len(preview) > 30 {
					preview = preview[:27] + "..."
				}
				s.WriteString(fmt.Sprintf("%s [ID: %s] Заметка: %s\n", cursor, secret.ID, preview))
			case *models.MetaBinary:
				s.WriteString(fmt.Sprintf("%s [ID: %s] Файл: %s (%s) | %d байт\n", cursor, secret.ID, v.Name, v.Type, v.Size))
			default:
				s.WriteString(fmt.Sprintf("%s [ID: %s] Данные неизвестного типа\n", cursor, secret.ID))
			}
		}
	}
	return s.String()
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
