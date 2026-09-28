package models

import (
	"errors"
	"time"
)

var (
	// ErrUserAlreadyExists определяет ошибку, когда логин занят.
	ErrUserAlreadyExists = errors.New("логин занят")
	// ErrUserNotFound определяет ошибку, когда пользователь не найден.
	ErrUserNotFound = errors.New("пользователь не найден")
)

// SecretType определяет тип данных.
type SecretType string

// Типы секретных данных, поддерживаемые приложением.
const (
	TypeUserData SecretType = "login/password"
	TypeText     SecretType = "text"
	TypeBinary   SecretType = "binary"
	TypeCard     SecretType = "card"
)

// User содержит данные о пользователе.
type User struct {
	ID           int
	Login        string
	PasswordHash string
}

// CipherData содержит зашифрованные данных для использования сервером.
type CipherData struct {
	ID        int        `json:"id,omitempty"`
	UserID    int        `json:"-"`
	Type      SecretType `json:"type"`
	Cipher    []byte     `json:"payload"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// LoginPassword определяет пары логин/пароль.
type LoginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// TextData определяет произвольный текст.
type TextData struct {
	Text string `json:"text"`
}

// BinaryData определяет произвольные файлы.
type BinaryData struct {
	Data []byte `json:"data"`
}

// Card определяет данные банковских карт.
type Card struct {
	Number   string `json:"number"`
	Date     string `json:"date"`
	UserName string `json:"user_name"`
	Key      string `json:"key"`
}
