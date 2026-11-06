package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/danilov-go/gophkeeper/internal/server/handler"
	"github.com/danilov-go/gophkeeper/internal/server/handler/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

func TestHandler_RegisterUser(t *testing.T) {
	tests := []struct {
		name      string
		body      interface{}
		mockSetup func(m *mocks.ServerService)
		expStatus int
	}{
		{
			name: "положительный тест",
			body: models.RegisterUser{Login: expLogin, Hash: expHash, Salt: expSalt},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Register", mock.Anything, expLogin, expHash, expSalt).
					Return(token, nil).Once()
			},
			expStatus: http.StatusOK,
		},
		{
			name:      "ошибка десериализации",
			body:      "invalid",
			mockSetup: func(m *mocks.ServerService) {},
			expStatus: http.StatusBadRequest,
		},
		{
			name:      "пустой логин или хэш",
			body:      models.RegisterUser{Login: "", Hash: ""},
			mockSetup: func(m *mocks.ServerService) {},
			expStatus: http.StatusBadRequest,
		},

		{
			name: "логин занят",
			body: models.RegisterUser{Login: expLogin, Hash: expHash, Salt: expSalt},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Register", mock.Anything, expLogin, expHash, expSalt).
					Return("", models.ErrUserAlreadyExists).Once()
			},
			expStatus: http.StatusConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := mocks.NewServerService(t)
			tt.mockSetup(serv)
			logger := zaptest.NewLogger(t)
			h := handler.NewHandlers(logger.Sugar(), serv)
			register := h.RegisterUser()
			var buf bytes.Buffer
			if strBody, ok := tt.body.(string); ok {
				buf.WriteString(strBody)
			} else {
				err := json.NewEncoder(&buf).Encode(tt.body)
				assert.NoError(t, err)
			}
			req := httptest.NewRequest(http.MethodPost, "/user/register", &buf)
			res := httptest.NewRecorder()
			register(res, req)
			assert.Equal(t, tt.expStatus, res.Code)
			if tt.expStatus == http.StatusOK {
				authHeader := res.Header().Get("Authorization")
				assert.Contains(t, authHeader, "Bearer "+token)
			}
		})
	}
}

func TestHandler_LoginUser(t *testing.T) {
	tests := []struct {
		name      string
		body      interface{}
		mockSetup func(m *mocks.ServerService)
		expStatus int
	}{
		{
			name: "положительный тест",
			body: map[string]string{"login": expLogin, "hash": expHash},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Login", mock.Anything, expLogin, expHash).
					Return(token, expSalt, nil).Once()
			},
			expStatus: http.StatusOK,
		},
		{
			name:      "ошибка десериализации",
			body:      "invalid",
			mockSetup: func(m *mocks.ServerService) {},
			expStatus: http.StatusBadRequest,
		},
		{
			name:      "пустой логин или хэш",
			body:      map[string]string{"login": "", "hash": ""},
			mockSetup: func(m *mocks.ServerService) {},
			expStatus: http.StatusBadRequest,
		},

		{
			name: "неверный логин или хэш",
			body: map[string]string{"login": expLogin, "hash": expHash},
			mockSetup: func(m *mocks.ServerService) {
				m.On("Login", mock.Anything, expLogin, expHash).
					Return("", "", models.ErrUserNotFound).Once()
			},
			expStatus: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := mocks.NewServerService(t)
			tt.mockSetup(serv)
			logger := zaptest.NewLogger(t)
			h := handler.NewHandlers(logger.Sugar(), serv)
			loginUser := h.LoginUser()
			var buf bytes.Buffer
			err := json.NewEncoder(&buf).Encode(tt.body)
			assert.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/user/login", &buf)
			res := httptest.NewRecorder()
			loginUser(res, req)
			assert.Equal(t, tt.expStatus, res.Code)
			if res.Code == http.StatusOK {
				assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
				assert.Contains(t, res.Header().Get("Authorization"), "Bearer "+token)
				var authSalt models.AuthSalt
				err := json.Unmarshal(res.Body.Bytes(), &authSalt)
				assert.NoError(t, err)
				assert.Equal(t, expSalt, authSalt.Salt)
			}
		})
	}
}

func TestHandler_GetSalt_Mock(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		mockSetup  func(m *mocks.ServerService)
		wantStatus int
	}{
		{
			name:  "положительный тест",
			query: "?login=" + expLogin,
			mockSetup: func(m *mocks.ServerService) {
				m.On("GetSalt", mock.Anything, expLogin).
					Return(expSalt).Once()
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "отсутствует логин",
			query:      "",
			mockSetup:  func(m *mocks.ServerService) {},
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := mocks.NewServerService(t)
			tt.mockSetup(serv)
			logger := zaptest.NewLogger(t)
			h := handler.NewHandlers(logger.Sugar(), serv)
			getSalt := h.GetSalt()
			req := httptest.NewRequest(http.MethodGet, "/user/salt"+tt.query, nil)
			res := httptest.NewRecorder()
			getSalt(res, req)
			assert.Equal(t, tt.wantStatus, res.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, "application/json", res.Header().Get("Content-Type"))
				var authSalt models.AuthSalt
				err := json.Unmarshal(res.Body.Bytes(), &authSalt)
				assert.NoError(t, err)
				assert.Equal(t, expSalt, authSalt.Salt)
			}
		})
	}
}

func TestHandler_AuthMiddleware_Mock(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		mockSetup  func(m *mocks.ServerService)
		expStatus  int
		expNext    bool
	}{
		{
			name:       "положительный тест",
			authHeader: "Bearer " + token,
			mockSetup: func(m *mocks.ServerService) {
				m.On("GetUserLogin", token).
					Return(models.AuthUser{ID: userID, Login: expLogin}, nil).Once()
			},
			expStatus: http.StatusOK,
			expNext:   true,
		},
		{
			name:       "пустой заголовок Authorization",
			authHeader: "",
			mockSetup:  func(m *mocks.ServerService) {},
			expStatus:  http.StatusUnauthorized,
			expNext:    false,
		},
		{
			name:       "невалидный токен",
			authHeader: token + "invalid",
			mockSetup: func(m *mocks.ServerService) {
				m.On("GetUserLogin", token+"invalid").
					Return(models.AuthUser{}, errors.New("token is invalid")).Once()
			},
			expStatus: http.StatusUnauthorized,
			expNext:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := mocks.NewServerService(t)
			tt.mockSetup(serv)
			logger := zaptest.NewLogger(t)
			h := handler.NewHandlers(logger.Sugar(), serv)
			next := false
			nextHandler := func(w http.ResponseWriter, r *http.Request, user models.AuthUser) {
				next = true
				assert.Equal(t, userID, user.ID)
				assert.Equal(t, expLogin, user.Login)
				w.WriteHeader(http.StatusOK)
			}
			auth := h.AuthMiddleware(nextHandler)
			req := httptest.NewRequest(http.MethodGet, "/api/update", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			res := httptest.NewRecorder()
			auth(res, req)
			assert.Equal(t, tt.expStatus, res.Code)
			assert.Equal(t, tt.expNext, next)
		})
	}
}
