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
	ID    int
	Login string
	Hash  string
	Salt  string
}

type RegisterUser struct {
	Login string `json:"login"`
	Hash  string `json:"hash"`
	Salt  string `json:"salt"`
}

type AuthSalt struct {
	Salt string `json:"salt"`
}

type LocalUser struct {
	Hash string
	Salt string
}
