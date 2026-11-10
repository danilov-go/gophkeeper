package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/server/handler"
	"github.com/danilov-go/gophkeeper/internal/server/handler/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

const (
	deviceID = "device"
	secretID = "test"
	expLogin = "login"
	expHash  = "password"
	expSalt  = "test_salt"
	userID   = 15
	token    = "jwt_secret_token"
)

func TestHandler_PullHandler(t *testing.T) {
	user := models.AuthUser{ID: userID, Login: expLogin}
	secret := []models.CipherData{{
		ID:        secretID,
		Type:      models.TypeCard,
		Version:   2,
		UpdatedAt: time.Time{},
	}}
	tests := []struct {
		name        string
		contentType string
		body        interface{}
		mockSetup   func(m *mocks.ServerService)
		expStatus   int
		expBody     string
	}{
		{
			name:        "положительный тест",
			contentType: "application/json",
			body:        models.SecretVersions{},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Pull", mock.Anything, user.ID, mock.Anything).
					Return(secret, nil).Once()
			},
			expStatus: http.StatusOK,
			expBody:   `[{"id":"test","type":"card","cipher":null,"version":2,"deleted":false,"updated_at":"0001-01-01T00:00:00Z"}]`,
		},
		{
			name:        "положительный тест (StatusNoContent)",
			contentType: "application/json",
			body:        models.SecretVersions{},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Pull", mock.Anything, user.ID, mock.Anything).
					Return([]models.CipherData{}, nil).Once()
			},
			expStatus: http.StatusNoContent,
		},
		{
			name:        "ошибка Content-Type",
			contentType: "text/plain",
			body:        "plain text",
			mockSetup:   func(m *mocks.ServerService) {},
			expStatus:   http.StatusUnsupportedMediaType,
		},
		{
			name:        "ошибка десериализации",
			contentType: "application/json",
			body:        "invalid",
			mockSetup:   func(m *mocks.ServerService) {},
			expStatus:   http.StatusBadRequest,
		},
		{
			name:        "внутренняя ошибка сервиса",
			contentType: "application/json",
			body:        models.SecretVersions{},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Pull", mock.Anything, user.ID, mock.Anything).
					Return(nil, errors.New("test error")).Once()
			},
			expStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := mocks.NewServerService(t)
			tt.mockSetup(serv)
			logger := zaptest.NewLogger(t)
			h := handler.NewHandlers(logger.Sugar(), serv)
			pull := h.PullHandler()
			var buf bytes.Buffer
			if strBody, ok := tt.body.(string); ok {
				buf.WriteString(strBody)
			} else {
				err := json.NewEncoder(&buf).Encode(tt.body)
				assert.NoError(t, err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/pull", &buf)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			res := httptest.NewRecorder()
			pull(res, req, user)
			assert.Equal(t, tt.expStatus, res.Code)
			if tt.expBody != "" {
				assert.JSONEq(t, tt.expBody, res.Body.String())
			}
		})
	}
}

func TestHandler_PushHandler(t *testing.T) {
	user := models.AuthUser{ID: userID, Login: expLogin}
	tests := []struct {
		name        string
		contentType string
		deviceID    string
		body        interface{}
		mockSetup   func(m *mocks.ServerService)
		expStatus   int
	}{
		{
			name:        "положительный тест",
			contentType: "application/json",
			deviceID:    deviceID,
			body: models.Changelogs{
				{SecretID: secretID, Secret: models.CipherData{ID: secretID, Version: 5}},
			},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Push", mock.Anything, user.ID, deviceID, mock.Anything).
					Return(nil).Once()
			},
			expStatus: http.StatusOK,
		},
		{
			name:        "отсутствует заголовок device-id",
			contentType: "application/json",
			deviceID:    "",
			body:        models.Changelogs{},
			mockSetup:   func(m *mocks.ServerService) {},
			expStatus:   http.StatusBadRequest,
		},
		{
			name:        "внутренняя ошибка сервисе",
			contentType: "application/json",
			deviceID:    deviceID,
			body:        models.Changelogs{},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Push", mock.Anything, user.ID, deviceID, mock.Anything).
					Return(errors.New("test error")).Once()
			},
			expStatus: http.StatusInternalServerError,
		},
		{
			name:        "ошибка десериализации",
			contentType: "application/json",
			deviceID:    deviceID,
			body:        "invalid",
			mockSetup:   func(m *mocks.ServerService) {},
			expStatus:   http.StatusBadRequest,
		},
		{
			name:        "ошибка Content-Type",
			contentType: "text/plain",
			deviceID:    deviceID,
			body:        models.Changelogs{},
			mockSetup:   func(m *mocks.ServerService) {},
			expStatus:   http.StatusUnsupportedMediaType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := mocks.NewServerService(t)
			tt.mockSetup(serv)
			logger := zaptest.NewLogger(t)
			h := handler.NewHandlers(logger.Sugar(), serv)
			push := h.PushHandler()
			var buf bytes.Buffer
			if strBody, ok := tt.body.(string); ok {
				buf.WriteString(strBody)
			} else {
				err := json.NewEncoder(&buf).Encode(tt.body)
				assert.NoError(t, err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/push", &buf)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			if tt.deviceID != "" {
				req.Header.Set("device-id", tt.deviceID)
			}
			res := httptest.NewRecorder()
			push(res, req, user)
			assert.Equal(t, tt.expStatus, res.Code)
		})
	}
}

func TestHandler_StreamHandler(t *testing.T) {
	user := models.AuthUser{ID: userID, Login: expLogin}
	serv := mocks.NewServerService(t)
	eventChan := make(chan models.SyncEvent, 1)
	cancelCalled := false
	mockCancel := func() { cancelCalled = true }
	serv.On("Event", mock.Anything, user.ID, deviceID).
		Return((<-chan models.SyncEvent)(eventChan), mockCancel).Once()
	logger := zaptest.NewLogger(t)
	h := handler.NewHandlers(logger.Sugar(), serv)
	stream := h.StreamHandler()
	reqCtx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil).WithContext(reqCtx)
	req.Header.Set("device-id", deviceID)
	res := httptest.NewRecorder()
	eventChan <- models.SyncEvent{Message: "sync"}
	done := make(chan bool)
	go func() {
		stream(res, req, user)
		done <- true
	}()
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("хендлер не завершил работу после отмены контекста")
	}
	assert.Equal(t, "text/event-stream", res.Header().Get("Content-Type"))
	assert.Equal(t, "keep-alive", res.Header().Get("Connection"))
	assert.Contains(t, res.Body.String(), "data: sync\n\n")
	assert.True(t, cancelCalled)
}
