// Package models содержит общие структуры данных и типы для синхронизации секретов.
package models

import (
	"time"
)

// CipherData содержит зашифрованные данных для передачи.
type CipherData struct {
	// ID содержит уникальный идентификатор секрета.
	ID string `json:"id,omitempty" example:"8f3b1442-7a2e-4b41-a6e5-2276537af353"`
	// UserID содержит идентификатор владельца секрета.
	UserID int `json:"-"`
	// Type определяет тип секретных данных
	Type SecretType `json:"type" example:"card" `
	// Cipher определяет зашифрованное содержимое секрета.
	Cipher []byte `json:"cipher" swaggertype:"string" example:"eyJudW1iZXIiOjQyNDIsImRhdGUiOiIxMi8yNiIsInVzZXJfbmFtZSI6IlVzZXJOYW1lIiwia2V5IjoiMTIzIn0="`
	// UpdatedAt содержит время последнего обновления записи.
	UpdatedAt time.Time `json:"updated_at" example:"2026-11-03T15:00:00Z"`
	// Version содержит текущую версию секрета.
	Version int `json:"version" example:"1"`
	// Deleted содержит флаг, указывающий на удаление секрета.
	Deleted bool `json:"deleted" example:"false"`
}

// LoginPassword определяет пары логин/пароль.
type LoginPassword struct {
	// Login содержит имя пользователя или email для входа.
	Login string `json:"login"`
	// Password содержит пароль от учетной записи.
	Password string `json:"password"`
	// SecretTOTP содержит ключ для генерации одноразовых кодов.
	SecretTOTP string `json:"secret_totp,omitempty"`
}

// TextData определяет произвольный текст.
type TextData struct {
	//Text содержит текст заметки.
	Text string `json:"text"`
}

// BinaryData определяет произвольные файлы.
type BinaryData struct {
	// Data определяет содержимое файла в виде массива байт.
	Data []byte `json:"data"`
}

// Card определяет данные банковских карт.
type Card struct {
	//Number содержит номер карты.
	Number string `json:"number"`
	//Date содержит срок действия карты.
	Date string `json:"date"`
	//UserName содержит имя и фамилию владельца карты.
	UserName string `json:"user_name"`
	//Key содержит CVC код.
	Key string `json:"key"`
}
