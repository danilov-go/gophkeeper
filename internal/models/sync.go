package models

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

type SecretVersions map[int]int
type Changelogs []Changelog
