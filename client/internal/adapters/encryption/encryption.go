package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"golang.org/x/crypto/argon2"
	"gophkeeper/client/internal/domain"
	"io"
)

type EncryptionService interface {
	// DeriveKey — мастер-пароль и соль = AES-256 ключ
	DeriveKey(masterPassword, salt []byte) ([]byte, error)

	// Encrypt — шифрует данные
	Encrypt(data, key []byte) ([]byte, error)

	// Decrypt — расшифровывает данные
	Decrypt(encryptedData, key []byte) ([]byte, error)
}

// DefaultEncryption — реализация по умолчанию
type DefaultEncryption struct{}

func NewEncryptionService() EncryptionService {
	return &DefaultEncryption{}
}

func (e *DefaultEncryption) DeriveKey(masterPassword, salt []byte) ([]byte, error) {

	if len(masterPassword) == 0 {
		return nil, domain.ErrMasterPasswordRequired
	}

	if len(salt) == 0 {
		salt := make([]byte, 16)
		rand.Read(salt)
	}

	key := argon2.IDKey(masterPassword, salt, 1, 64*1024, 4, 32)
	return key, nil
}

func (e *DefaultEncryption) Encrypt(data, key []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, domain.ErrEncryptionFailed
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (e *DefaultEncryption) Decrypt(encryptedData, key []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, domain.ErrDecryptionFailed
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
