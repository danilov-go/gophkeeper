package models

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims содержит полезную нагрузку JWT-токена.
type Claims struct {
	// ID содержит уникальный идентификатор пользователя.
	ID int `json:"id"`
	// Login содержит логин пользователя.
	Login string `json:"login"`
	jwt.RegisteredClaims
}

// AuthUser содержит данные об авторизованном пользователе.
type AuthUser struct {
	// ID содержит уникальный идентификатор пользователя.
	ID int
	// Login содержит логин пользователя.
	Login string
}

// TokenExp определяет время жизни JWT-токена.
const TokenExp = time.Hour * 24

var (
	// ErrUserAlreadyExists определяет ошибку, когда логин занят.
	ErrUserAlreadyExists = errors.New("логин занят")
	// ErrUserNotFound определяет ошибку, когда пользователь не найден.
	ErrUserNotFound = errors.New("пользователь не найден")
)

// User содержит данные о пользователе.
type User struct {
	// ID содержит уникальный идентификатор пользователя.
	ID int
	// Login содержит логин пользователя.
	Login string
	// Hash содержит хеш пароля.
	Hash string
	// Salt содержит соль для хеширования.
	Salt string
}

// RegisterUser содержит данные для регистрации нового пользователя.
type RegisterUser struct {
	// Login содержит логин пользователя.
	Login string `json:"login"`
	// Hash содержит хеш пароля.
	Hash string `json:"hash"`
	// Salt содержит соль.
	Salt string `json:"salt"`
}

// AuthSalt содержит соль пользователя при аутентификации.
type AuthSalt struct {
	// Salt содержит соль.
	Salt string `json:"salt"`
}

// LocalUser содержит локальные данные авторизации пользователя.
type LocalUser struct {
	// Hash содержит хеш пароля.
	Hash string
	// Salt содержит соль.
	Salt string
}
