package sender

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/stretchr/testify/assert"
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

func TestHTTPSender_Register(t *testing.T) {
	tests := []struct {
		name     string
		resp     func(w http.ResponseWriter, r *http.Request)
		expToken string
		expErr   string
	}{
		{
			name: "положительный тест",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Authorization", "Bearer "+token)
				w.WriteHeader(http.StatusOK)
			},
			expToken: token,
			expErr:   "",
		},
		{
			name: "логин занят",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
			},
			expToken: "",
			expErr:   "логин занят",
		},
		{
			name: "oшибка сервера",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expToken: "",
			expErr:   "статус ответа от сервера",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.resp))
			defer server.Close()
			cfg := config.ConfigClient{}
			logger := zaptest.NewLogger(t).Sugar()
			sender := NewHTTPSender(cfg, logger, deviceID, "")
			sender.client.SetBaseURL(server.URL)
			err := sender.Register(context.Background(), expLogin, expHash, expSalt)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expToken, sender.token)
			}
		})
	}
}

func TestHTTPSender_Auth(t *testing.T) {
	tests := []struct {
		name     string
		resp     func(w http.ResponseWriter, r *http.Request)
		expToken string
		expErr   string
	}{
		{
			name: "положительный тест",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Authorization", "Bearer "+token)
				w.WriteHeader(http.StatusOK)
			},
			expToken: token,
			expErr:   "",
		},
		{
			name: "ошибка авторизации",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			expToken: "",
			expErr:   "статус ответа от сервера",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.resp))
			defer server.Close()
			cfg := config.ConfigClient{}
			logger := zaptest.NewLogger(t).Sugar()
			sender := NewHTTPSender(cfg, logger, deviceID, "")
			sender.client.SetBaseURL(server.URL)
			err := sender.Auth(context.Background(), expLogin, expHash)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expToken, sender.token)
			}
		})
	}
}

func TestHTTPSender_Sync(t *testing.T) {
	tests := []struct {
		name     string
		expToken string
		resp     func(w http.ResponseWriter, r *http.Request)
		expLen   int
		expErr   string
	}{
		{
			name:     "положительный тест",
			expToken: token,
			resp: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/pull" {
					w.Header().Set("Content-Type", "application/json")
					cipherData := []models.CipherData{
						{UserID: userID, ID: secretID, Type: models.TypeCard},
						{UserID: userID, ID: secretID + "1", Type: models.TypeBinary},
					}
					_ = json.NewEncoder(w).Encode(cipherData)
				}
			},
			expLen: 2,
			expErr: "",
		},
		{
			name:     "ошибка сервера",
			expToken: token,
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expLen: 0,
			expErr: "статус ответа от сервера",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.resp))
			defer server.Close()
			cfg := config.ConfigClient{}
			logger := zaptest.NewLogger(t).Sugar()
			sender := NewHTTPSender(cfg, logger, deviceID, "")
			sender.client.SetBaseURL(server.URL)
			sender.token = tt.expToken
			versions := map[string]models.VersionTime{secretID: {Version: 1, UpdatedAt: time.Now()}}
			cipherData, err := sender.Sync(context.Background(), versions, nil)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Len(t, cipherData, tt.expLen)
			}
		})
	}
}

func TestHTTPSender_GetSalt(t *testing.T) {
	tests := []struct {
		name    string
		resp    func(w http.ResponseWriter, r *http.Request)
		expSalt string
		expErr  string
	}{
		{
			name: "положительный тест",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(models.AuthSalt{Salt: expSalt})
			},
			expSalt: expSalt,
			expErr:  "",
		},
		{
			name: "Ошибка сервера",
			resp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expSalt: "",
			expErr:  "сервер вернул статус",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.resp))
			defer server.Close()
			cfg := config.ConfigClient{}
			logger := zaptest.NewLogger(t).Sugar()
			sender := NewHTTPSender(cfg, logger, deviceID, "")
			sender.client.SetBaseURL(server.URL)
			salt, err := sender.GetSalt(context.Background(), expLogin)
			if tt.expErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expSalt, salt)
			}
		})
	}
}

func TestHTTPSender_Logout(t *testing.T) {
	sender := &HTTPSender{token: token}
	sender.Logout()
	assert.Empty(t, sender.token)
}
