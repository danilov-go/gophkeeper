package crypto_test

import (
	"testing"

	"github.com/danilov-go/gophkeeper/internal/crypto"
	"github.com/stretchr/testify/assert"
)

const (
	password = "password"
)

var key = []byte("12345678912345678912345678912345")

func TestGenerateSalt(t *testing.T) {
	salt1, err := crypto.GenerateSalt()
	assert.NoError(t, err)
	assert.Len(t, salt1, 32)
	salt2, err := crypto.GenerateSalt()
	assert.NoError(t, err)
	assert.NotEqual(t, salt1, salt2)
}

func TestGenerateKey(t *testing.T) {
	salt, err := crypto.GenerateSalt()
	assert.NoError(t, err)
	tests := []struct {
		name     string
		saltHex  string
		password string
		wantErr  bool
	}{
		{
			name:     "положительный тест",
			saltHex:  salt,
			password: password,
			wantErr:  false,
		},
		{
			name:     "пустой пароль",
			saltHex:  salt,
			password: "",
			wantErr:  true,
		},
		{
			name:     "пустая соль",
			saltHex:  "",
			password: password,
			wantErr:  true,
		},
		{
			name:     "не валидная соль",
			saltHex:  "invalid",
			password: password,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, hash, err := crypto.GenerateKey(tt.saltHex, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, key)
				assert.Empty(t, hash)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.Len(t, key, 32)
		})
	}
}

func TestEncrypt_Decrypt(t *testing.T) {
	secretData := []byte("secret data")
	cipherBody, err := crypto.Encrypt(key, secretData)
	assert.NoError(t, err)
	assert.NotEmpty(t, cipherBody)
	decryptedData, err := crypto.Decrypt(key, cipherBody)
	assert.NoError(t, err)
	assert.Equal(t, secretData, decryptedData)
	key = []byte("short-key")
	_, err = crypto.Encrypt(key, secretData)
	assert.Error(t, err)
}

func TestHash_Check(t *testing.T) {
	hash, err := crypto.Hash(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	ok, err := crypto.Check(password, hash)
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = crypto.Check("invalid_password", hash)
	assert.NoError(t, err)
	assert.False(t, ok)
	ok, err = crypto.Check(password, "invalid_hash")
	assert.Error(t, err)
	assert.False(t, ok)
}
