package domain

import "errors"

var (
	// ErrInvalidCredentials — неверный email или пароль
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserAlreadyExists — пользователь с таким email уже существует
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrUserNotFound — пользователь не найден
	ErrUserNotFound = errors.New("user not found")
)
