package sender

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danilov-go/gophkeeper/internal/config"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/go-resty/resty/v2"
)

type log interface {
	Errorw(msg string, keysAndValues ...any)
}

type HTTPSender struct {
	client   *resty.Client
	logger   log
	key      []byte
	token    string
	deviceID string
	mu       sync.RWMutex
}

// NewHTTPSender создает экземпляр HTTPSender.
func NewHTTPSender(cfg config.ConfigClient, l log, deviceID string) *HTTPSender {
	client := resty.New().
		SetTimeout(time.Second * 10).
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
		client:   client,
		logger:   l,
		deviceID: deviceID,
	}
}

// Register отправляет запрос на регистрацию нового пользователя.
func (s *HTTPSender) Register(ctx context.Context, login, hash, salt string) error {
	body := models.RegisterUser{
		Login: login,
		Hash:  hash,
		Salt:  salt,
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(body).
		Post("/user/register")
	if err != nil {
		s.logger.Errorw("ошибка при регистрации", "error", err)
		return err
	}
	if resp.StatusCode() == http.StatusConflict {
		return errors.New("логин занят")
	}
	if resp.IsError() {
		s.logger.Errorw("сервер вернул ошибку", "status", resp.Status())
		return fmt.Errorf("статус ответа от сервера: %s", resp.Status())
	}
	token := resp.Header().Get("Authorization")
	if token == "" {
		return errors.New("токен не передан сервером")
	}
	s.mu.Lock()
	s.token = strings.TrimPrefix(token, "Bearer ")
	s.mu.Unlock()
	return nil
}

// Auth выполняет аутентификацию пользователя на сервере и сохраняет JWT-токен.
func (s *HTTPSender) Auth(ctx context.Context, login, hash string) error {
	body := models.RegisterUser{
		Login: login,
		Hash:  hash,
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(body).
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
	s.mu.Lock()
	s.token = strings.TrimPrefix(token, "Bearer ")
	s.mu.Unlock()
	return nil
}

func (s *HTTPSender) Sync(ctx context.Context, versions map[int]int, changelogs []models.Changelog) ([]models.CipherData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	token := s.token
	s.mu.RUnlock()
	if token == "" {
		return nil, errors.New("клиент не авторизован")
	}
	cipherData, err := s.pull(ctx, versions, token)
	if err != nil {
		return nil, err
	}
	if len(changelogs) > 0 {
		err = s.push(ctx, changelogs, token)
		if err != nil {
			return nil, err
		}
	}
	return cipherData, nil
}

func (s *HTTPSender) pull(ctx context.Context, versions map[int]int, token string) ([]models.CipherData, error) {
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetHeader("Content-Type", "application/json").
		SetBody(versions).
		Post("/api/pull")
	if err != nil {
		s.logger.Errorw("ошибка при выполнении запроса", "error", err)
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		var cipherData []models.CipherData
		if err := json.Unmarshal(resp.Body(), &cipherData); err != nil {
			s.logger.Errorw("ошибка десериализации", "error", err)
			return nil, err
		}
		return cipherData, nil
	}
	if resp.StatusCode() == http.StatusNoContent {
		return nil, nil
	}
	s.logger.Errorw("сервер вернул ошибку", "status", resp.Status())
	return nil, fmt.Errorf("статус ответа от сервера: %s", resp.Status())
}

func (s *HTTPSender) push(ctx context.Context, changelogs []models.Changelog, token string) error {
	if len(changelogs) == 0 {
		return nil
	}
	resp, err := s.client.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetHeader("Content-Type", "application/json").
		SetHeader("device-id", s.deviceID).
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

	return nil
}

func (s *HTTPSender) Logout() {
	s.token = ""
}

func (s *HTTPSender) ServerEvent(ctx context.Context, runSync func()) {
	client := &http.Client{
		Timeout: 0,
	}
	go func() {
		for {
			if err := ctx.Err(); err != nil {
				return
			}
			s.mu.RLock()
			token := s.token
			s.mu.RUnlock()
			if token == "" {
				time.Sleep(time.Second * 1)
				continue
			}
			req, err := http.NewRequestWithContext(ctx, "GET", s.client.BaseURL+"/api/event", nil)
			if err != nil {
				time.Sleep(time.Second * 2)
				continue
			}
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Accept", "text/event-stream")
			req.Header.Set("device-id", s.deviceID)
			resp, err := client.Do(req)
			if err != nil {
				s.logger.Errorw("не удалось подключиться к серверу оповещений", "error", err)
				time.Sleep(time.Second * 5)
				continue
			}
			reader := bufio.NewReader(resp.Body)
			for {
				if err := ctx.Err(); err != nil {
					resp.Body.Close()
					return
				}
				s.mu.RLock()
				nextToken := s.token
				s.mu.RUnlock()
				if nextToken != token {
					break
				}
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				if strings.HasPrefix(line, "data: sync") {
					runSync()
				}
			}
			resp.Body.Close()
			time.Sleep(time.Second * 2)
		}
	}()
}

func (s *HTTPSender) GetSalt(ctx context.Context, login string) (string, error) {
	var req models.AuthSalt
	resp, err := s.client.R().
		SetContext(ctx).
		SetQueryParam("login", login).
		SetResult(&req).
		Get("/user/salt")
	if err != nil {
		s.logger.Errorw("ошибка при выполнении запроса", "error", err)
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf("сервер вернул статус: %s", resp.Status())
	}
	return req.Salt, nil
}
