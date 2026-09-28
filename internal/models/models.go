package models

import "errors"

// User содержит данные о пользователе.
type User struct {
	// ID содержит уникальный идентификатор пользователя.
	ID int
	// Login содержит логин пользователя.
	Login string
	// PasswordHash содержит хешированный пароль пользователя.
	PasswordHash string
}

var (
	// ErrOrderAlreadyUploadedBySameUser определяет ошибку, когда номер заказа уже был загружен этим пользователем.
	ErrOrderAlreadyUploadedBySameUser = errors.New("номер заказа уже был загружен этим пользователем")
	// ErrOrderAlreadyUploadedByOtherUser определяет ошибку, когда номер заказа уже был загружен другим пользователем.
	ErrOrderAlreadyUploadedByOtherUser = errors.New("номер заказа уже был загружен другим пользователем")
	// ErrUserAlreadyExists определяет ошибку, когда логин занят.
	ErrUserAlreadyExists = errors.New("логин занят")
	// ErrUserNotFound определяет ошибку, когда пользователь не найден.
	ErrUserNotFound = errors.New("пользователь не найден")
)
