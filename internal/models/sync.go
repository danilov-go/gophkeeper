package models

import "time"

// SyncAction содержит тип действия при синхронизации данных.
type SyncAction int

// Доступные типы действий при синхронизации.
const (
	ActionSkip SyncAction = iota
	ActionUpdate
	ActionDelete
	ActionConflict
)

// Changelog содержит запись в журнале изменений конкретного секрета.
type Changelog struct {
	// ID содержит идентификатор записи.
	ID int `json:"id" example:"15"`
	// SecretID содержит строковый идентификатор секрета.
	SecretID string `json:"secret_id" example:"8f3b1442-7a2e-4b41-a6e5-2276537af353"`
	// Secret содержит зашифрованные данные секрета.
	Secret CipherData `json:"secret"`
}

// VersionTime содержит номер версии и время последнего обновления секрета.
type VersionTime struct {
	// Version содержит номер версии.
	Version int `json:"version" example:"1"`
	// UpdatedAt содержит время последнего обновления.
	UpdatedAt time.Time `json:"updated_at" example:"2026-11-03T15:00:00Z"`
}

// SecretVersions содержит карту версий секретов.
type SecretVersions map[string]VersionTime

// Changelogs содержит список изменений для пакетной обработки.
type Changelogs []Changelog

// SyncEvent содержит данные о событии синхронизации для уведомления устройств.
type SyncEvent struct {
	// UserID содержит идентификатор пользователя.
	UserID int `example:"1"`
	// DeviceID содержит идентификатор устройства пользователя.
	DeviceID string `example:"device1"`
	// Message содержит сообщение события.
	Message string `example:"sync"`
}

// Conflict содержит информацию о конфликтующих версиях данных между клиентом и сервером.
type Conflict struct {
	// ID содержит идентификатор секрета.
	ID string
	// Local содержит локальную расшифрованную версию секрета.
	Local Secret
	// Server содержит серверную расшифрованную версию секрета.
	Server Secret
}
