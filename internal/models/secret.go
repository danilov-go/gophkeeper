package models

import (
	"encoding/json"
	"fmt"
)

// SecretType определяет тип данных.
type SecretType string

// Типы секретных данных, поддерживаемые приложением.
const (
	TypeLoginPassword SecretType = "login/password"
	TypeText          SecretType = "text"
	TypeBinary        SecretType = "binary"
	TypeCard          SecretType = "card"
)

type SecretData interface {
	SecretType() SecretType
}

func (*LoginPassword) SecretType() SecretType {
	return TypeLoginPassword
}

func (*TextData) SecretType() SecretType {
	return TypeText
}

func (*BinaryData) SecretType() SecretType {
	return TypeBinary
}

func (*Card) SecretType() SecretType {
	return TypeCard
}

func NewSecretData(secretType SecretType, body []byte) (SecretData, error) {
	switch secretType {
	case TypeCard:
		var data Card
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case TypeLoginPassword:
		var data LoginPassword
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case TypeText:
		var data TextData
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		return &data, nil
	case TypeBinary:
		var data BinaryData
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		return &data, nil
	default:
		return nil, fmt.Errorf("неизвестный тип: %s", secretType)
	}
}

type Secret struct {
	ID   int        `json:"id"`
	Type SecretType `json:"type"`
	Data SecretData `json:"data"`
	Meta MetaData   `json:"meta"`
}
