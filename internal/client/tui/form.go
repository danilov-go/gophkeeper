package tui

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/danilov-go/gophkeeper/internal/models"
)

type FormOption struct {
	Label string
	Data  string
	Limit int
	Hide  bool
}

type FormModel struct {
	service  SecretService
	sType    models.SecretType
	secretID int
	option   []FormOption
	inputs   []textinput.Model
	focus    int
	saved    bool
	err      error
}

func NewFormModel(srv SecretService, sType models.SecretType) FormModel {
	option := newOptions(sType)
	inputs := newInputs(option)
	return FormModel{
		service:  srv,
		sType:    sType,
		secretID: 0,
		option:   option,
		inputs:   inputs,
	}
}

func NewEditForm(srv SecretService, secret models.Secret) FormModel {
	option := newOptions(secret.Type)
	inputs := newInputs(option)
	switch secret.Type {
	case models.TypeCard:
		meta := secret.Meta.(*models.MetaCard)
		card := secret.Data.(*models.Card)
		inputs[0].SetValue(meta.Name)
		inputs[1].SetValue(meta.BankName)
		inputs[2].SetValue(card.Number)
		inputs[3].SetValue(card.Date)
		inputs[4].SetValue(card.UserName)
		inputs[5].SetValue(card.Key)
	case models.TypeLoginPassword:
		meta := secret.Meta.(*models.MetaLoginPassword)
		lp := secret.Data.(*models.LoginPassword)
		inputs[0].SetValue(meta.Name)
		inputs[1].SetValue(meta.URL)
		inputs[2].SetValue(lp.Login)
		inputs[3].SetValue(lp.Password)
	case models.TypeText:
		meta := secret.Meta.(*models.MetaText)
		txt := secret.Data.(*models.TextData)
		inputs[0].SetValue(meta.Name)
		inputs[1].SetValue(txt.Text)
	case models.TypeBinary:
		meta := secret.Meta.(*models.MetaBinary)
		inputs[0].SetValue(meta.Name)
		inputs[1].SetValue(strconv.FormatInt(meta.Size, 10))
		inputs[2].SetValue(meta.Type)
		inputs[3].SetValue("")
	}
	return FormModel{
		service:  srv,
		sType:    secret.Type,
		secretID: secret.ID,
		option:   option,
		inputs:   inputs,
	}
}

func newOptions(sType models.SecretType) []FormOption {
	switch sType {
	case models.TypeCard:
		return []FormOption{
			{Label: "Описание", Data: "Введите описание ...", Limit: 50},
			{Label: "Название банка", Data: "Сбербанк", Limit: 50},
			{Label: "Номер карты", Data: "XXXX XXXX XXXX XXXX", Limit: 19},
			{Label: "Срок действия", Data: "MM/YY", Limit: 5},
			{Label: "Владелец", Data: "IVAN DANILOV", Limit: 30},
			{Label: "Код (CVV)", Data: "•••", Limit: 4, Hide: true},
		}
	case models.TypeLoginPassword:
		return []FormOption{
			{Label: "Описание", Data: "Введите описание ...", Limit: 40},
			{Label: "URL", Data: "http://yandex.ru", Limit: 40},
			{Label: "Логин / Email", Data: "Введите логин ...", Limit: 40},
			{Label: "Пароль", Data: "Введите пароль ...", Limit: 40, Hide: true},
		}
	case models.TypeText:
		return []FormOption{
			{Label: "Описание", Data: "Введите описание ...", Limit: 50},
			{Label: "Текст", Data: "Введите текст...", Limit: 500},
		}
	case models.TypeBinary:
		return []FormOption{
			{Label: "Название", Data: "file.txt", Limit: 50},
			{Label: "Размер", Data: "", Limit: 50},
			{Label: "Тип", Data: "", Limit: 50},
			{Label: "Путь к файлу на диске", Data: "/path/to/file.txt", Limit: 255},
		}
	}
	return nil
}

func newInputs(configs []FormOption) []textinput.Model {
	inputs := make([]textinput.Model, len(configs))
	for i, cfg := range configs {
		inputs[i] = textinput.New()
		inputs[i].Placeholder = cfg.Data
		inputs[i].CharLimit = cfg.Limit
		if cfg.Hide {
			inputs[i].EchoMode = textinput.EchoPassword
		}
	}
	inputs[0].Focus()
	return inputs
}

func (m FormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "down", "enter":
			if m.focus == len(m.inputs)-1 {
				return m, m.saveCmd()
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
		if msg == "success" {
			m.saved = true
		}
	case error:
		m.err = msg
	}
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m FormModel) View() string {
	var s strings.Builder
	if m.secretID > 0 {
		s.WriteString(fmt.Sprintf("Редактирование записи [ID: %d]: %s\n\n", m.secretID, m.sType))
	} else {
		s.WriteString(fmt.Sprintf("Добавление записи: %s\n\n", m.sType))
	}
	if m.err != nil {
		s.WriteString(fmt.Sprintf("Ошибка: %v\n\n", m.err))
	}
	if m.saved {
		s.WriteString("Запись сохранена!\n\n")
		return s.String()
	}
	for i := range m.inputs {
		if m.sType == models.TypeBinary && (i == 1 || i == 2) {
			s.WriteString(fmt.Sprintf(" %s: %s \n", m.option[i].Label, m.inputs[i].Value()))
			continue
		}
		s.WriteString(fmt.Sprintf(" %s: %s\n", m.option[i].Label, m.inputs[i].View()))
	}
	return s.String()
}

func (m FormModel) switchFocus() (FormModel, tea.Cmd) {
	if m.sType == models.TypeBinary {
		if m.focus == 1 || m.focus == 2 {
			if m.focus == 1 {
				m.focus = 3
			} else {
				m.focus = 0
			}
		}
	}
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

func (m *FormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i < len(m.inputs); i++ {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m FormModel) saveCmd() tea.Cmd {
	return func() tea.Msg {
		if m.inputs[0].Value() == "" {
			return errors.New("добавьте описание")
		}
		var data models.SecretData
		var meta models.MetaData
		switch m.sType {
		case models.TypeCard:
			meta = &models.MetaCard{
				Name:     m.inputs[0].Value(),
				BankName: m.inputs[1].Value(),
			}
			data = &models.Card{
				Number:   m.inputs[2].Value(),
				Date:     m.inputs[3].Value(),
				UserName: m.inputs[4].Value(),
				Key:      m.inputs[5].Value(),
			}
		case models.TypeLoginPassword:
			meta = &models.MetaLoginPassword{
				Name: m.inputs[0].Value(),
				URL:  m.inputs[1].Value(),
			}
			data = &models.LoginPassword{
				Login:    m.inputs[2].Value(),
				Password: m.inputs[3].Value(),
			}
		case models.TypeText:
			meta = &models.MetaText{
				Name: m.inputs[0].Value(),
			}
			data = &models.TextData{
				Text: m.inputs[1].Value(),
			}
		case models.TypeBinary:
			filePath := m.inputs[3].Value()
			if filePath == "" && m.secretID == 0 {
				return errors.New("путь к файлу не может быть пустым")
			}
			var dataFile []byte
			var ext string
			var size int64
			if filePath != "" {
				var err error
				dataFile, err = m.service.GetFile(filePath)
				if err != nil {
					return err
				}
				ext = filepath.Ext(filePath)
				size = int64(len(dataFile))
			}
			meta = &models.MetaBinary{
				Name: m.inputs[0].Value(),
				Size: size,
				Type: ext,
			}
			data = &models.BinaryData{
				Data: dataFile,
			}
		}
		var err error
		if m.secretID > 0 {
			err = m.service.Update(context.Background(), m.secretID, meta, data)
		} else {
			_, err = m.service.Save(context.Background(), meta, data)
		}
		if err != nil {
			return err
		}
		return "success"
	}
}
