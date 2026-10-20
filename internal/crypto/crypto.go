package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
	sizeSalt     = 16
)

func GenerateSalt() (string, error) {
	random := make([]byte, sizeSalt)
	_, err := rand.Read(random)
	if err != nil {
		return "", err
	}
	salt := hex.EncodeToString(random)
	return salt, nil
}

// GenerateKey генерирует криптографический ключ из мастер-пароля и логина с помощью Argon2id.
func GenerateKey(saltHex string, password string) ([]byte, string, error) {
	if password == "" || saltHex == "" {
		return nil, "", errors.New("соль или пароль не заданы")
	}
	saltBytes, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, "", err
	}
	key := argon2.IDKey([]byte(password), saltBytes, argonTime, argonMemory, uint8(argonThreads), argonKeyLen)
	serverSalt := append(saltBytes, []byte("-server-auth")...)
	hashBytes := argon2.IDKey(key, serverSalt, argonTime, argonMemory, uint8(argonThreads), argonKeyLen)
	hash := base64.RawStdEncoding.EncodeToString(hashBytes)
	return key, hash, nil
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

// Hash возвращает argon2 хеш пароля.
func Hash(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	encodedHash := fmt.Sprintf(`$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s`,
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash)
	return encodedHash, nil
}

// Check проверяет argon2 хеш пароля.
func Check(password, encodedHash string) (bool, error) {
	parsHash := strings.Split(encodedHash, `$`)
	if len(parsHash) != 6 {
		return false, errors.New("invalid hash format")
	}
	if parsHash[1] != "argon2id" {
		return false, errors.New("incompatible argon2 variant")
	}
	var version int
	_, err := fmt.Sscanf(parsHash[2], "v=%d", &version)
	if err != nil || version != argon2.Version {
		return false, err
	}
	var memory, iterations, parallelism uint32
	_, err = fmt.Sscanf(parsHash[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, err
	}
	if parallelism > 255 {
		return false, errors.New("invalid parallelism parameter: value too large")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parsHash[4])
	if err != nil {
		return false, err
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(parsHash[5])
	if err != nil {
		return false, err
	}
	otherHash := argon2.IDKey([]byte(password), salt, iterations, memory, uint8(parallelism), uint32(len(expectedHash)))
	if subtle.ConstantTimeCompare(expectedHash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}
