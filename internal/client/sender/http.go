package sender

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/go-resty/resty/v2"
)

type HTTPSender struct {
	client  *resty.Client
	logger  log
	key     []byte
	token   string
	storage Storage
}

// NewHTTPSender создает экземпляр HTTPSender.
func NewHTTPSender(cfg config.ConfigClient, l log, s Storage) *HTTPSender {
	client := resty.New().
		SetTimeout(time.Second * 3).
		SetBaseURL("http://" + cfg.Net.String()).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(5 * time.Second)
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return true
		}
		if r.StatusCode() >= http.StatusInternalServerError {
			return true
		}
		return false
	})
	return &HTTPSender{
		client:  client,
		logger:  l,
		storage: s,
	}
}

// Register отправляет запрос на регистрацию нового пользователя.
func (s *HTTPSender) Register(ctx context.Context, login, password string) error {
	body := models.LoginPassword{
		Login:    login,
		Password: password,
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/user/register")
	if err != nil {
		s.logger.Errorw("ошибка при регистрации", "error", err)
		return err
	}
	if resp.IsError() {
		s.logger.Errorw("сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	token := resp.Header().Get("Authorization")
	if token == "" {
		return errors.New("токен не передан сервером")
	}
	s.token = strings.TrimPrefix(token, "Bearer ")
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
		Post("/user/login")
	if err != nil {
		s.logger.Errorw("ошибка при авторизации", "error", err)
		return err
	}
	if resp.IsError() {
		s.logger.Errorw("сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	token := resp.Header().Get("Authorization")
	if token == "" {
		return errors.New("токен не передан сервером")
	}
	s.token = strings.TrimPrefix(token, "Bearer ")
	return nil
}

func (s *HTTPSender) Sync(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.token == "" {
		return nil
	}
	err := s.pull(ctx)
	if err != nil {
		return err
	}
	err = s.push(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *HTTPSender) pull(ctx context.Context) error {
	versions, err := s.storage.GetVersion(ctx)
	if err != nil {
		return err
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.token).
		SetHeader("Content-Type", "application/json").
		SetBody(versions).
		Post("/api/pull")
	if err != nil {
		s.logger.Errorw("ошибка при выполнении запроса", "error", err)
		return err
	}
	if resp.StatusCode() == http.StatusOK {
		var cipherData []models.CipherData
		if err := json.Unmarshal(resp.Body(), &cipherData); err == nil && len(cipherData) > 0 {
			err = s.storage.SyncUpdate(ctx, cipherData)
			if err != nil {
				return err
			}
		}
	} else if resp.StatusCode() != http.StatusNoContent {
		s.logger.Errorw("сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	return nil
}

func (s *HTTPSender) push(ctx context.Context) error {
	changelogs, err := s.storage.GetChangelog()
	if err != nil {
		return err
	}
	if len(changelogs) == 0 {
		return nil
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(s.token).
		SetHeader("Content-Type", "application/json").
		SetBody(changelogs).
		Post("/api/push")
	if err != nil {
		s.logger.Errorw("ошибка при выполнении запроса", "error", err)
		return err
	}
	if resp.IsError() {
		s.logger.Errorw("сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	id := changelogs[len(changelogs)-1].ID
	s.storage.DeleteChangelog(id)
	return nil
}
