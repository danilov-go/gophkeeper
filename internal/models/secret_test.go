package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSecretData(t *testing.T) {
	tests := []struct {
		name       string
		secretType SecretType
		jsonBody   string
		expErr     bool
		expData    SecretData
	}{
		{
			name:       "положительный тест(TypeCard)",
			secretType: TypeCard,
			jsonBody:   `{"number":"4242","date":"11/26","user_name":"user_name","key":"123"}`,
			expErr:     false,
			expData:    &Card{Number: "4242", Date: "11/26", UserName: "user_name", Key: "123"},
		},
		{
			name:       "положительный тест(TypeLoginPassword)",
			secretType: TypeLoginPassword,
			jsonBody:   `{"login":"login","password":"password","secret_totp":"secret_totp"}`,
			expErr:     false,
			expData:    &LoginPassword{Login: "login", Password: "password", SecretTOTP: "secret_totp"},
		},
		{
			name:       "положительный тест(TypeText)",
			secretType: TypeText,
			jsonBody:   `{"text":"text"}`,
			expErr:     false,
			expData:    &TextData{Text: "text"},
		},
		{
			name:       "положительный тест(TypeBinary)",
			secretType: TypeBinary,
			jsonBody:   `{"data":"ZGF0YQ=="}`,
			expErr:     false,
			expData:    &BinaryData{Data: []byte("data")},
		},
		{
			name:       "передаче неизвестного типа",
			secretType: SecretType("unknown"),
			jsonBody:   `{"name":"unknown"}`,
			expErr:     true,
			expData:    nil,
		},
		{
			name:       "битый JSON",
			secretType: TypeCard,
			jsonBody:   `invalid`,
			expErr:     true,
			expData:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := NewSecretData(tt.secretType, []byte(tt.jsonBody))
			if tt.expErr {
				assert.Error(t, err)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
				assert.Equal(t, tt.expData, data)
				assert.Equal(t, tt.secretType, data.SecretType())
			}
		})
	}
}
