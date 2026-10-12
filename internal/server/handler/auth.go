package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// LoginHandlerFunc определяет тип функции-обработчика для запросов, требующих авторизации.
type LoginHandlerFunc func(w http.ResponseWriter, r *http.Request, user AuthUser)

// GetUserLogin проверяет валидность токена и возвращает данные пользователя.
func GetUserLogin(tokenString, key string) (AuthUser, error) {
	var user AuthUser
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(key), nil
	})
	if err != nil {
		return AuthUser{}, err
	}
	if !token.Valid {
		return AuthUser{}, errors.New("token is not valid")
	}
	user = AuthUser{
		ID:    claims.ID,
		Login: claims.Login,
	}
	return user, nil
}

// AuthMiddleware проверяет наличие JWT-токена в заголовке Authorization и валидирует его.
func AuthMiddleware(key string, h LoginHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(token, "Bearer ")
		user, err := GetUserLogin(tokenString, key)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		h(w, r, user)
	}
}
