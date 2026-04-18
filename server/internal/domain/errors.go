package domain

import "errors"

var (
	// ErrInvalidCredentials — общая ошибка для логина, неверный email или пароль
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserAlreadyExists — для регистрации, пользователь с таким email уже существует
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrUserNotFound — пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrTokenInvalid, ErrTokenExpired — для JWT
	ErrTokenInvalid = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")

	// Общие
	ErrNotFound = errors.New("not found")
	ErrInternal = errors.New("internal error")
)
