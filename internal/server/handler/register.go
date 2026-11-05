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

// LoginPassword используется для десериализации учетных данных при входе.
type LoginPassword struct {
	Login string `json:"login" example:"user"`
	Hash  string `json:"hash" example:"$argon2id$v=19$m=65536,t=3,p=2$J/gjOEdE2a3CReVybKk0KA$w060F3W2YnO2S/tL5v7rSwGgAn9/k08Y6M1o4L3s7uE"`
}

func decode[T any](w http.ResponseWriter, r *http.Request) (T, error) {
	var body T
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024*1024)
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		return body, err
	}
	return body, nil
}

// RegisterUser возвращает обработчик для регистрации нового пользователя.
//
// @Summary      Регистрация нового пользователя
// @Description  Создает новый аккаунт пользователя и возвращает токен авторизации
// @Tags         user
// @Accept       json
// @Produce      json
// @Param        body  body      models.RegisterUser  true  "Данные регистрации (login, hash, salt)"
// @Success      200   "Успешная регистрация, токен возвращается в заголовке Authorization"
// @Failure      400   {string}  string "Неверные входные данные или пустой логин/пароль"
// @Failure      409   {string}  string "Логин уже занят другим пользователем"
// @Failure      500   {string}  string "Внутренняя ошибка сервера"
// @Router       /user/register [post]
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
//
// @Summary Аутентификация пользователя
// @Description Проверяет логин и хэш пароля. Возвращает JWT-токен в заголовке Authorization и соль пользователя в теле ответа.
// @Tags user
// @Accept json
// @Produce json
// @Param body body LoginPassword true "Логин и хеш пароля"
// @Success 200 {object} models.AuthSalt "Соль для дальнейшей аутентификации на клиенте" {header} string Authorization "Bearer JWT-токен"
// @Failure 400 {string} string "Неверные данные или пустые обязательные поля"
// @Failure 401 {string} string "Пользователь не найден или неверный пароль"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /user/login [post]
func (h *Handler) LoginUser() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, err := decode[LoginPassword](w, r)
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
			h.logger.Errorw("ошибка в процессе входа пользователя", "error", err, "login", user.Login)

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

// GetSalt возвращает соль пользователя по его логину для последующей аутентификации.
//
// @Summary      Получить соль пользователя по логину
// @Description  Используется клиентом перед авторизацией, чтобы узнать соль пользователя
// @Tags         user
// @Produce      json
// @Param        login  query     string                true  "Логин пользователя"
// @Success      200    {object}  models.AuthSalt       "Соль пользователя"
// @Failure      400    {string}  string "Отсутствует или пустой login"
// @Failure      500    {string}  string  "Внутренняя ошибка сервера"
// @Router       /user/salt [get]
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
