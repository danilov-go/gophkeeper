package models

import (
	"encoding/json"
	"fmt"
)

// MetaData возвращает тип метаданных секрета.
type MetaData interface {
	SecretType() SecretType
}

// MetaCard определяет метаданные для типа Card.
type MetaCard struct {
	// Name содержит описание, вводимое пользователем.
	Name string `json:"name"`
	// BankName содержит наименование банка.
	BankName string `json:"bank_name"`
}

// MetaLoginPassword определяет метаданные для типа LoginPassword.
type MetaLoginPassword struct {
	// Name содержит описание, вводимое пользователем.
	Name string `json:"name"`
	// URL содержит адрес веб-страницы авторизации.
	URL string `json:"url"`
}

// MetaText определяет метаданные для типа Text.
type MetaText struct {
	// Name содержит описание, вводимое пользователем.
	Name string `json:"name"`
}

// MetaBinary определяет метаданные для типа Binary.
type MetaBinary struct {
	// Name содержит название файла с расширением.
	Name string `json:"name"`
	//Size содержит размер файла.
	Size int64 `json:"size"`
	//Type содержит расширение файла.
	Type string `json:"type"`
}

func (*MetaCard) SecretType() SecretType          { return TypeCard }
func (*MetaLoginPassword) SecretType() SecretType { return TypeLoginPassword }
func (*MetaText) SecretType() SecretType          { return TypeText }
func (*MetaBinary) SecretType() SecretType        { return TypeBinary }

// NewMetaData создает метаданные секрета из JSON на основе их типа.
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
