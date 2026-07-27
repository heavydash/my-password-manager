package domain

import "errors"

var (
	// Общие ошибки
	ErrInvalidInput           = errors.New("invalid input")
	ErrInvalidMasterPassword  = errors.New("invalid or weak master password")
	ErrMasterPasswordRequired = errors.New("master password is required")

	// Ошибки сети / API
	ErrConnectionFailed = errors.New("connection to server failed")
	ErrUnauthorized     = errors.New("unauthorized - please login again")
	ErrServerError      = errors.New("server internal error")

	// Ошибки шифрования
	ErrEncryptionFailed = errors.New("encryption failed")
	ErrDecryptionFailed = errors.New("decryption failed")
)
