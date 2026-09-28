package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/go-resty/resty/v2"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

// HTTPSender отвечает за сетевое взаимодействие с сервером по протоколу HTTPS.
type HTTPSender struct {
	client *resty.Client
	logger log
	key    string
	token  string
}

// NewHTTPSender создает экземпляр HTTPSender.
func NewHTTPSender(serverURL string, key string, l log) *HTTPSender {
	client := resty.New().
		SetTimeout(time.Second * 3).
		SetBaseURL("https://" + serverURL).
		SetRetryCount(3).
		SetRetryAfter(func(c *resty.Client, r *resty.Response) (time.Duration, error) {
			attempt := 1
			if r != nil && r.Request != nil {
				attempt = r.Request.Attempt
			}
			return time.Duration(1+2*(attempt-1)) * time.Second, nil
		})
	return &HTTPSender{
		client: client,
		logger: l,
		key:    key,
	}
}

// Register отправляет запрос на регистрацию нового пользователя.
func (s *HTTPSender) Register(ctx context.Context, login, password string) error {
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
func (s *HTTPSender) Auth(ctx context.Context, login, password string) error {
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
