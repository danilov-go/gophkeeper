package models

import "time"

type ActionType string

const (
	ActionCreate ActionType = "CREATE"
	ActionUpdate ActionType = "UPDATE"
	ActionDelete ActionType = "DELETE"
)

type Changelog struct {
	ID       int        `json:"id"`
	Action   ActionType `json:"action"`
	SecretID int        `json:"secret_id"`
	Secret   CipherData `json:"secret"`
}
type VersionTime struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SecretVersions map[int]VersionTime
type Changelogs []Changelog

type SyncEvent struct {
	UserID   int
	DeviceID string
	Message  string
}

type SyncAction int

const (
	ActionSkip SyncAction = iota
	ActionUpdateSecret
	ActionConflict
)

type Conflict struct {
	ID     int
	Local  Secret
	Server Secret
}
