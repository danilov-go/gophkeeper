package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danilov-go/gophkeeper/internal/models"
)

// LoginHandlerFunc определяет тип функции-обработчика для запросов, требующих авторизации.
type LoginHandlerFunc func(w http.ResponseWriter, r *http.Request, user models.AuthUser)

type loginPassword struct {
	Login string `json:"login"`
	Hash  string `json:"hash"`
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return body, err
	}
	return body, nil
}

// RegisterUser возвращает обработчик для регистрации нового пользователя.
func (h *Handler) RegisterUser() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, err := decode[models.RegisterUser](w, r)
		if err != nil {
			h.logger.Errorw("ошибка десилиризации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if user.Login == "" || user.Hash == "" {
			h.logger.Errorw("пустой логин или пароль", "login", user.Login, "password", user.Hash)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		signedToken, err := h.service.Register(ctx, user.Login, user.Hash, user.Salt)
		if err != nil {
			if errors.Is(err, models.ErrUserAlreadyExists) {
				http.Error(w, "логин занят", http.StatusConflict)
				return
			}
			h.logger.Errorw("ошибка аутентификации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Authorization", "Bearer "+signedToken)
		w.WriteHeader(http.StatusOK)
	})
}

// LoginUser возвращает обработчик для аутентификации существующего пользователя.
func (h *Handler) LoginUser() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, err := decode[loginPassword](w, r)
		if err != nil {
			h.logger.Errorw("ошибка десериализации", "error", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if user.Login == "" || user.Hash == "" {
			h.logger.Errorw("пустой логин или пароль", "login", user.Login)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		signedToken, salt, err := h.service.Login(ctx, user.Login, user.Hash)
		if err != nil {
			if errors.Is(err, models.ErrUserNotFound) {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			h.logger.Errorw("ошибка аутентификации", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Authorization", "Bearer "+signedToken)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := models.AuthSalt{
			Salt: salt,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			h.logger.Errorw("ошибка сериализации ответа с солью", "error", err)
			return
		}

	})
}

func (h *Handler) GetSalt() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		login := r.URL.Query().Get("login")
		if login == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		salt := h.service.GetSalt(ctx, login)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := models.AuthSalt{
			Salt: salt,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// AuthMiddleware проверяет наличие JWT-токена в заголовке Authorization и валидирует его.
func (h *Handler) AuthMiddleware(next LoginHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(token, "Bearer ")
		user, err := h.service.GetUserLogin(tokenString)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next(w, r, user)
	}
}
