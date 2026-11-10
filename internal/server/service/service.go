// Package service содержит бизнес-логику сервера.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/danilov-go/gophkeeper/internal/crypto"
	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// Storage определяет методы для взаимодействия с хранилищем.
type Storage interface {
	SaveUser(ctx context.Context, login, hash, salt string) (int, error)
	GetUser(ctx context.Context, login string) (models.User, error)
	GetSecrets(ctx context.Context, userID int, versions models.SecretVersions) ([]models.CipherData, error)
	UpdateSecrets(ctx context.Context, userID int, changelogs models.Changelogs) error
}

// Publisher определяет управление подписками и рассылки уведомлений.
type Publisher interface {
	Register(userID int, deviceID string) (<-chan models.SyncEvent, func())
	Notify(event models.SyncEvent)
}

// ServerService определяет методы бизнес-логики сервера.
type ServerService struct {
	storage  Storage
	jwtKey   string
	notifier Publisher
}

// NewServerService создает новый экземпляр ServerService.
func NewServerService(s Storage, jwtKey string, notifier Publisher) *ServerService {
	return &ServerService{
		storage:  s,
		jwtKey:   jwtKey,
		notifier: notifier,
	}
}

// BuildJWTString генерирует токен на основе моделей данных.
func (s *ServerService) BuildJWTString(id int, login string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(models.TokenExp)),
		},
		ID:    id,
		Login: login,
	})
	return token.SignedString([]byte(s.jwtKey))
}

// Register хеширует пароль и регистрирует нового пользователя.
func (s *ServerService) Register(ctx context.Context, login, hash, salt string) (string, error) {
	serverHash, err := crypto.Hash(hash)
	if err != nil {
		return "", err
	}
	id, err := s.storage.SaveUser(ctx, login, serverHash, salt)
	if err != nil {
		if errors.Is(err, models.ErrUserAlreadyExists) {
			return "", models.ErrUserAlreadyExists
		}
		return "", err
	}

	return s.BuildJWTString(id, login)
}

// Login проверяет учетные данные пользователя и возвращает токен и соль.
func (s *ServerService) Login(ctx context.Context, login, hash string) (string, string, error) {
	user, err := s.storage.GetUser(ctx, login)
	if err != nil {
		return "", "", err
	}
	ok, err := crypto.Check(hash, user.Hash)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", models.ErrUserNotFound
	}
	token, err := s.BuildJWTString(user.ID, login)
	if err != nil {
		return "", "", err
	}
	return token, user.Salt, nil
}

// GetSalt возвращает соль пользователя или генерирует псевдослучайную соль, если пользователь не найден.
func (s *ServerService) GetSalt(ctx context.Context, login string) string {
	user, err := s.storage.GetUser(ctx, login)
	if err != nil {
		hash := sha256.Sum256([]byte(login))
		return hex.EncodeToString(hash[:16])
	}
	return user.Salt
}

// GetUserLogin проверяет валидность токена и возвращает данные пользователя.
func (s *ServerService) GetUserLogin(tokenString string) (models.AuthUser, error) {
	var user models.AuthUser
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtKey), nil
	})
	if err != nil {
		return models.AuthUser{}, err
	}
	if !token.Valid {
		return models.AuthUser{}, errors.New("token is not valid")
	}
	user = models.AuthUser{
		ID:    claims.ID,
		Login: claims.Login,
	}
	return user, nil
}

// Pull запрашивает из хранилища обновленные данные секретов на основе переданных версий.
func (s *ServerService) Pull(ctx context.Context, userID int, versions models.SecretVersions) ([]models.CipherData, error) {
	return s.storage.GetSecrets(ctx, userID, versions)
}

// Push сохраняет изменения секретов и уведомляет другие устройства пользователя о синхронизации.
func (s *ServerService) Push(ctx context.Context, userID int, deviceID string, changelogs models.Changelogs) error {
	err := s.storage.UpdateSecrets(ctx, userID, changelogs)
	if err != nil {
		return err
	}
	s.notifier.Notify(models.SyncEvent{
		UserID:   userID,
		DeviceID: deviceID,
		Message:  "sync",
	})
	return nil
}

// Event регистрирует устройство для прослушивания событий синхронизации.
func (s *ServerService) Event(ctx context.Context, userID int, deviceID string) (<-chan models.SyncEvent, func()) {
	return s.notifier.Register(userID, deviceID)
}
