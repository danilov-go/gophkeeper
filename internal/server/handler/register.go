package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danilov-go/gophkeeper/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonSaltLen = 16
	argonKeyLen  = 32
)

type loginPassword struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// BuildJWTString создает строку JWT-токена для указанного пользователя.
func BuildJWTString(id int, login, key string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		ID:    id,
		Login: login,
	})
	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		return "", err
	}
	return tokenString, nil
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

func decode(w http.ResponseWriter, r *http.Request) (loginPassword, error) {
	var user loginPassword
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		return loginPassword{}, err
	}
	return user, nil
}

// RegisterUser возвращает обработчик для регистрации нового пользователя.
func (h *Handler) RegisterUser(key string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, err := decode(w, r)
		if err != nil {
			h.logger.Errorw("ошибка десилиризации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if user.Login == "" || user.Password == "" {
			h.logger.Errorw("пустой логин или пароль", "login", user.Login, "password", user.Password)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		hashStringPassword, err := Hash(user.Password)
		if err != nil {
			h.logger.Errorw("ошибка хеширования пароля", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		id, err := h.storage.SaveUser(ctx, user.Login, hashStringPassword)
		if err != nil {
			if errors.Is(err, models.ErrUserAlreadyExists) {
				h.logger.Errorw("пользователь с таким именем уже существует", "error", err)
				http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
				return
			}
			h.logger.Errorw("ошибка сохранения пользователя", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		signedToken, err := BuildJWTString(id, user.Login, key)
		if err != nil {
			h.logger.Errorw("ошибка аутентификации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Authorization", "Bearer "+signedToken)
		w.WriteHeader(http.StatusOK)
	})
}

// LoginUser возвращает обработчик для аутентификации существующего пользователя.
func (h *Handler) LoginUser(key string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, err := decode(w, r)
		if err != nil {
			h.logger.Errorw("ошибка десериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if user.Login == "" || user.Password == "" {
			h.logger.Errorw("пустой логин или пароль", "login", user.Login)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		store, err := h.storage.GetUser(ctx, user.Login)
		if err != nil {
			h.logger.Errorw("неверный логин или пароль", "error", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		ok, err := Check(user.Password, store.PasswordHash)
		if err != nil || !ok {
			h.logger.Errorw("пароли не совпадают")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		signedToken, err := BuildJWTString(store.ID, user.Login, key)
		if err != nil {
			h.logger.Errorw("ошибка аутентификации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Authorization", "Bearer "+signedToken)
		w.WriteHeader(http.StatusOK)
	})
}
