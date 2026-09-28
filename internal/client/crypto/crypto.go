package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
)

// GenerateKey генерирует криптографический ключ из мастер-пароля и логина с помощью Argon2id.
func GenerateKey(password string, login string) []byte {
	hash := sha256.New()
	hash.Write([]byte(login))
	salt := hash.Sum(nil)
	return argon2.IDKey([]byte(password), salt, argonTime, argonMemory, uint8(argonThreads), argonKeyLen)
}

// Encrypt шифрует данные с помощью ключа.
func Encrypt(key, body []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	cipherBody := gcm.Seal(nonce, nonce, body, nil)
	return cipherBody, nil
}

// Decrypt расшифровывает данные с помощью ключа.
func Decrypt(key, cipherBody []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(cipherBody) < nonceSize {
		return nil, io.ErrUnexpectedEOF
	}
	nonce := cipherBody[:nonceSize]
	actualCipherText := cipherBody[nonceSize:]
	body, err := gcm.Open(nil, nonce, actualCipherText, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}
