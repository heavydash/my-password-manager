package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"gophkeeper/client/internal/domain"
)

type KeyManager struct {
	enc EncryptionService
}

func NewKeyManager() *KeyManager {
	return &KeyManager{
		enc: NewEncryptionService(),
	}
}

// GenerateSalt создаёт уникальную соль
func (km *KeyManager) GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// DeriveKeyFromMasterPassword выводит ключ из мастер-пароля и соли
func (km *KeyManager) DeriveKeyFromMasterPassword(masterPassword string, salt []byte) ([]byte, error) {
	if masterPassword == "" {
		return nil, domain.ErrMasterPasswordRequired
	}
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	return km.enc.DeriveKey([]byte(masterPassword), salt)
}

// SaltToString конвертирует соль в строку для сохранения
func SaltToString(salt []byte) string {
	return base64.StdEncoding.EncodeToString(salt)
}

// StringToSalt восстанавливает соль из строки
func StringToSalt(saltStr string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(saltStr)
}

// Encrypt - обёртка над EncryptionService
func (km *KeyManager) Encrypt(data []byte, key []byte) ([]byte, error) {
	return km.enc.Encrypt(data, key)
}

// Decrypt - обёртка над EncryptionService
func (km *KeyManager) Decrypt(encryptedData []byte, key []byte) ([]byte, error) {
	return km.enc.Decrypt(encryptedData, key)
}
