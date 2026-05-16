package domain

import "errors"

var (
	// Auth errors
	// ErrInvalidCredentials — общая ошибка для логина, неверный email или пароль
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserAlreadyExists — для регистрации, пользователь с таким email уже существует
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrEmailRequired     = errors.New("email is required")

	// OAuth errors
	ErrUnsupportedOAuthProvider = errors.New("unsupported oauth provider")
	ErrOAuthUnavailable         = errors.New("oauth unavailable") // ← новая
	ErrInvalidOAuthState        = errors.New("invalid state or expired")
	ErrOAuthCodeRequired        = errors.New("code is required")

	// ErrUserNotFound — пользователь не найден
	ErrUserNotFound = errors.New("user not found")

	// ErrTokenInvalid, ErrTokenExpired — для JWT
	ErrTokenInvalid = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")

	// Secret errors
	// ErrInvalidInput - неправильные вводимые данные
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrInternal     = errors.New("internal error")
)
