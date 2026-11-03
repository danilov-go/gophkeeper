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
	Login string `json:"login" example:"user"`
	// Hash содержит хеш пароля.
	Hash string `json:"hash" example:"$argon2id$v=19$m=65536,t=3,p=2$J/gjOEdE2a3CReVybKk0KA$w060F3W2YnO2S/tL5v7rSwGgAn9/k08Y6M1o4L3s7uE"`
	// Salt содержит соль.
	Salt string `json:"salt" example:"8f3b14427a2e4b41a6e52276537af353"`
}

// AuthSalt содержит соль пользователя при аутентификации.
type AuthSalt struct {
	// Salt содержит соль.
	Salt string `json:"salt" example:"8f3b14427a2e4b41a6e52276537af353"`
}

// LocalUser содержит локальные данные авторизации пользователя.
type LocalUser struct {
	// Hash содержит хеш пароля.
	Hash string
	// Salt содержит соль.
	Salt string
}
