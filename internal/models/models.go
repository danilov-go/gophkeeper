package models

import (
	"time"
)

type SecretID struct {
	ID int `json:"id"`
}

// CipherData содержит зашифрованные данных для передачи.
type CipherData struct {
	ID        string     `json:"id,omitempty"`
	UserID    int        `json:"-"`
	Type      SecretType `json:"type"`
	Cipher    []byte     `json:"cipher"`
	UpdatedAt time.Time  `json:"updated_at"`
	Version   int        `json:"version"`
	Deleted   bool       `json:"deleted"`
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
