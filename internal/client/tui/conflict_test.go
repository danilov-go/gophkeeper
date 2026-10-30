package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/danilov-go/gophkeeper/internal/client/tui/mocks"
	"github.com/danilov-go/gophkeeper/internal/models"
)

const (
	conflict1 = "conflict1"
	conflict2 = "conflict2"
)

func TestConflictModel_Init(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(s *mocks.SecretService)
		expMsg    tea.Msg
	}{
		{
			name: "положительный тест",
			setupMock: func(s *mocks.SecretService) {
				s.On("GetConflicts", mock.Anything).Return([]models.Conflict{{ID: conflict1}}, nil).Once()
			},
			expMsg: []models.Conflict{{ID: conflict1}},
		},
		{
			name: "ошибка при получении списка конфликтов",
			setupMock: func(s *mocks.SecretService) {
				s.On("GetConflicts", mock.Anything).Return(nil, errTest).Once()
			},
			expMsg: errTest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			tt.setupMock(service)
			m := NewConflictModel(service)
			cmd := m.Init()
			assert.NotNil(t, cmd)
			assert.Equal(t, tt.expMsg, cmd())
		})
	}
}

func TestConflictModel_Update(t *testing.T) {
	conflicts := []models.Conflict{
		{ID: conflict1},
		{ID: conflict2},
	}
	tests := []struct {
		name       string
		initCursor int
		conflicts  []models.Conflict
		msg        tea.Msg
		setupMock  func(s *mocks.SecretService)
		cursor     int
		expErr     error
		wantCmd    bool
	}{
		{
			name:   "получение сообщения об ошибке",
			msg:    errTest,
			expErr: errTest,
		},
		{
			name:       "нажатие клавиши down",
			initCursor: 0,
			conflicts:  conflicts,
			msg:        tea.KeyMsg{Type: tea.KeyDown},
			cursor:     1,
		},
		{
			name:       "нажатие клавиши up",
			initCursor: 1,
			conflicts:  conflicts,
			msg:        tea.KeyMsg{Type: tea.KeyUp},
			cursor:     0,
		},
		{
			name:       "выбор опции 1",
			initCursor: 0,
			conflicts:  conflicts,
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")},
			setupMock: func(s *mocks.SecretService) {
				s.On("FixConflict", mock.Anything, conflict1, true).Return(nil).Once()
				s.On("GetConflicts", mock.Anything).Return(conflicts, nil).Maybe()
			},
			cursor:  0,
			wantCmd: true,
		},
		{
			name:       "выбор опции 2",
			initCursor: 1,
			conflicts:  conflicts,
			msg:        tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")},
			setupMock: func(s *mocks.SecretService) {
				s.On("FixConflict", mock.Anything, conflict2, false).Return(nil).Once()
				s.On("GetConflicts", mock.Anything).Return(conflicts, nil).Maybe()
			},
			cursor:  1,
			wantCmd: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			if tt.setupMock != nil {
				tt.setupMock(service)
			}
			m := NewConflictModel(service)
			m.conflicts = tt.conflicts
			m.cursor = tt.initCursor
			msg := tt.msg
			if key, ok := tt.msg.(tea.KeyMsg); ok {
				if string(key.Runes) == "up" || key.Type == tea.KeyUp {
					msg = tea.KeyMsg{Type: tea.KeyUp}
				} else if string(key.Runes) == "down" || key.Type == tea.KeyDown {
					msg = tea.KeyMsg{Type: tea.KeyDown}
				}
			}
			updatedModel, cmd := m.Update(msg)
			model := updatedModel.(ConflictModel)
			assert.Equal(t, tt.cursor, model.cursor)
			assert.Equal(t, tt.expErr, model.err)
			if tt.wantCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
		})
	}
}

func TestConflictModel_View(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		conflicts []models.Conflict
		text      []string
	}{
		{
			name:      "отображение ошибки загрузки",
			err:       errTest,
			conflicts: nil,
			text:      []string{"Ошибка загрузки конфликтов: test error"},
		},
		{
			name:      "отображение пустого списка конфликтов",
			err:       nil,
			conflicts: []models.Conflict{},
			text:      []string{"Отложенных конфликтов синхронизации не обнаружено"},
		},
		{
			name: "отображение списка конфликтов",
			err:  nil,
			conflicts: []models.Conflict{
				{
					ID: "secretID",
					Local: models.Secret{
						Type: models.TypeText,
						Data: &models.TextData{
							Text: "local",
						},
						Meta: &models.MetaText{},
					},
					Server: models.Secret{
						Type: models.TypeText,
						Data: &models.TextData{
							Text: "server",
						},
						Meta: &models.MetaText{},
					},
				},
			},
			text: []string{
				"Разрешение конфликтов синхронизации",
				"> Конфликт в секрете ID secretID",
				"Версия сервера",
				"server",
				"Локальная версия",
				"local",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := mocks.NewSecretService(t)
			service.On("GetAll", mock.Anything, mock.Anything).Return([]models.Secret{}, nil).Maybe()
			m := NewConflictModel(service)
			m.err = tt.err
			m.conflicts = tt.conflicts
			m.cursor = 0
			text := m.View()
			for _, expText := range tt.text {
				assert.Contains(t, text, expText)
			}
		})
	}
}
