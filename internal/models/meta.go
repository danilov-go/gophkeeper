package models

import (
	"encoding/json"
	"fmt"
)

type MetaData interface {
	SecretType() SecretType
}

type MetaCard struct {
	Name     string `json:"name"`
	BankName string `json:"bank_name"`
}

type MetaLoginPassword struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type MetaText struct {
	Name string `json:"name"`
}

type MetaBinary struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

func (*MetaCard) SecretType() SecretType {
	return TypeCard
}

func (*MetaLoginPassword) SecretType() SecretType {
	return TypeLoginPassword
}

func (*MetaText) SecretType() SecretType {
	return TypeText
}

func (*MetaBinary) SecretType() SecretType {
	return TypeBinary
}

func NewMetaData(secretType SecretType, body []byte) (MetaData, error) {
	switch secretType {
	case TypeCard:
		var meta MetaCard
		if err := json.Unmarshal(body, &meta); err != nil {
			return nil, err
		}
		return &meta, nil
	case TypeLoginPassword:
		var meta MetaLoginPassword
		if err := json.Unmarshal(body, &meta); err != nil {
			return nil, err
		}
		return &meta, nil
	case TypeText:
		var meta MetaText
		if err := json.Unmarshal(body, &meta); err != nil {
			return nil, err
		}
		return &meta, nil
	case TypeBinary:
		var meta MetaBinary
		if err := json.Unmarshal(body, &meta); err != nil {
			return nil, err
		}
		return &meta, nil
	default:
		return nil, fmt.Errorf("неизвестный тип метаданных: %s", secretType)
	}
}
