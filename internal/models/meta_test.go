package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMetaData(t *testing.T) {
	tests := []struct {
		name       string
		secretType SecretType
		jsonBody   string
		expErr     bool
		expMeta    MetaData
	}{
		{
			name:       "положительный тест(TypeCard)",
			secretType: TypeCard,
			jsonBody:   `{"name":"name","bank_name":"bank_name"}`,
			expErr:     false,
			expMeta:    &MetaCard{Name: "name", BankName: "bank_name"},
		},
		{
			name:       "положительный тест(TypeLoginPassword)",
			secretType: TypeLoginPassword,
			jsonBody:   `{"name":"name","url":"url"}`,
			expErr:     false,
			expMeta:    &MetaLoginPassword{Name: "name", URL: "url"},
		},
		{
			name:       "положительный тест(TypeText)",
			secretType: TypeText,
			jsonBody:   `{"name":"name"}`,
			expErr:     false,
			expMeta:    &MetaText{Name: "name"},
		},
		{
			name:       "положительный тест(TypeBinary)",
			secretType: TypeBinary,
			jsonBody:   `{"name":"name","size":123,"type":"type"}`,
			expErr:     false,
			expMeta:    &MetaBinary{Name: "name", Size: 123, Type: "type"},
		},
		{
			name:       "передаче неизвестного типа",
			secretType: SecretType("unknown"),
			jsonBody:   `{"name":"unknown"}`,
			expErr:     true,
			expMeta:    nil,
		},
		{
			name:       "битый JSON",
			secretType: TypeCard,
			jsonBody:   `invalid`,
			expErr:     true,
			expMeta:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta, err := NewMetaData(tt.secretType, []byte(tt.jsonBody))
			if tt.expErr {
				assert.Error(t, err)
				assert.Nil(t, meta)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, meta)
				assert.Equal(t, tt.expMeta, meta)
				assert.Equal(t, tt.secretType, meta.SecretType())
			}
		})
	}
}
