package models

import "time"

type SyncAction int

const (
	ActionSkip SyncAction = iota
	ActionUpdate
	ActionDelete
	ActionConflict
)

type Changelog struct {
	ID       int        `json:"id"`
	SecretID string     `json:"secret_id"`
	Secret   CipherData `json:"secret"`
}

type VersionTime struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SecretVersions map[string]VersionTime
type Changelogs []Changelog

type SyncEvent struct {
	UserID   int
	DeviceID string
	Message  string
}

type Conflict struct {
	ID     string
	Local  Secret
	Server Secret
}
