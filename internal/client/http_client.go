package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/danilov-go/gophkeeper/internal/client/crypto"
	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/go-resty/resty/v2"
)

type HTTPClient struct {
	client  *resty.Client
	logger  log
	key     []byte
	token   string
	storage Storage
}

// NewHTTPSender создает экземпляр HTTPSender.
func NewHTTPClient(cfg config.ConfigClient, l log, s Storage, cryptoKey []byte) *HTTPClient {
	client := resty.New().
		SetTimeout(time.Second * 3).
		SetBaseURL("https://" + cfg.Net.String()).
		SetRetryCount(3).
		SetRetryAfter(func(c *resty.Client, r *resty.Response) (time.Duration, error) {
			attempt := 1
			if r != nil && r.Request != nil {
				attempt = r.Request.Attempt
			}
			return time.Duration(1+2*(attempt-1)) * time.Second, nil
		})
	return &HTTPClient{
		client:  client,
		logger:  l,
		key:     cryptoKey,
		storage: s,
	}
}

// Register отправляет запрос на регистрацию нового пользователя.
func (s *HTTPClient) Register(ctx context.Context, login, password string) error {
	reqBody := models.LoginPassword{
		Login:    login,
		Password: password,
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(reqBody).
		Post("/api/user/register")
	if err != nil {
		s.logger.Errorw("ошибка при регистрации", "error", err)
		return err
	}
	if resp.IsError() {
		s.logger.Errorw("при регистрации сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	token := resp.Header().Get("Authorization")
	if token == "" {
		return errors.New("токен не передан сервером")
	}
	s.token = token
	return nil
}

// Auth выполняет аутентификацию пользователя на сервере и сохраняет JWT-токен.
func (s *HTTPClient) Auth(ctx context.Context, login, password string) error {
	reqBody := models.LoginPassword{
		Login:    login,
		Password: password,
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(reqBody).
		Post("/api/user/login")
	if err != nil {
		s.logger.Errorw("ошибка при авторизации", "error", err)
		return err
	}
	if resp.IsError() {
		s.logger.Errorw("при авторизации сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	token := resp.Header().Get("Authorization")
	if token == "" {
		return errors.New("токен не передан сервером")
	}
	s.token = token
	return nil
}

func (s *HTTPClient) SaveSecret(ctx context.Context, secret models.SecretData) error {
	secretType := secret.SecretType()
	body, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	cipherBody, err := crypto.Encrypt([]byte(s.key), body)
	if err != nil {
		return err
	}
	ciferData := models.CipherData{
		Type:      secretType,
		Cipher:    cipherBody,
		UpdatedAt: time.Now(),
	}
	localID, err := s.storage.Save(ctx, ciferData)
	if err != nil {
		s.logger.Errorw("ошибка сохранения в БД", "error", err)
		return err
	}
	var serverResponse models.SecretID
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.token).
		SetBody(ciferData).
		SetResult(&serverResponse).
		Post("/api/secret")
	if err != nil {
		s.logger.Errorw("ошибка запроса", "error", err)
		return nil
	}
	if resp.IsError() {
		s.logger.Errorw("сервер отклонил запрос", "status", resp.StatusCode())
		return errors.New("сервер отклонил запрос")
	}
	err = s.storage.UpdateID(ctx, localID, serverResponse.ID)
	if err != nil {
		s.logger.Errorw("ошибка обновления статуса синхронизации", "error", err)
		return err
	}
	return nil
}

func (s *HTTPClient) GetSecret(ctx context.Context, id int) (models.SecretData, error) {
	var cipherData models.CipherData
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.token).
		SetPathParam("id", fmt.Sprintf("%d", id)).
		SetResult(&cipherData).
		Get("/api/secret/{id}")
	if err != nil {
		s.logger.Errorw("ошибка запроса", "error", err)
		return nil, err
	}
	if resp.IsError() {
		s.logger.Errorw("сервер вернул ошибку при взятии секрета", "status", resp.StatusCode())
		return nil, fmt.Errorf("сервер вернул ошибку со статусом: %d", resp.StatusCode())
	}
	body, err := crypto.Decrypt([]byte(s.key), cipherData.Cipher)
	if err != nil {
		return nil, err
	}
	data, err := models.NewSecretData(cipherData.Type, body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetAllSecrets запрашивает с сервера все секреты пользователя,
func (s *HTTPClient) GetAllSecrets(ctx context.Context) ([]models.CipherData, error) {
	var secrets []models.CipherData
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.token).
		SetResult(&secrets).
		Get("/api/secret")
	if err != nil {
		s.logger.Errorw("ошибка запроса", "error", err)
		return nil, fmt.Errorf("сервер недоступен для синхронизации: %w", err)
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return nil, nil
		}
		s.logger.Errorw("ошибка сервера при синхронизации", "status", resp.StatusCode())
		return nil, errors.New("ошибка сервера при синхронизации")
	}
	err = s.storage.SaveAll(ctx, secrets)
	if err != nil {
		s.logger.Errorw("ошибка сохранения в БД", "error", err)
		return nil, err
	}
	return secrets, nil
}
