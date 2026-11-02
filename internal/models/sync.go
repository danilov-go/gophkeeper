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
	ID int `json:"id"`
	// SecretID содержит строковый идентификатор секрета.
	SecretID string `json:"secret_id"`
	// Secret содержит зашифрованные данные секрета.
	Secret CipherData `json:"secret"`
}

// VersionTime содержит номер версии и время последнего обновления секрета.
type VersionTime struct {
	// Version содержит номер версии.
	Version int `json:"version"`
	// UpdatedAt содержит время последнего обновления.
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretVersions содержит карту версий секретов.
type SecretVersions map[string]VersionTime

// Changelogs содержит список изменений для пакетной обработки.
type Changelogs []Changelog

// SyncEvent содержит данные о событии синхронизации для уведомления устройств.
type SyncEvent struct {
	// UserID содержит идентификатор пользователя.
	UserID int
	// DeviceID содержит идентификатор устройства пользователя.
	DeviceID string
	// Message содержит сообщение события.
	Message string
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
