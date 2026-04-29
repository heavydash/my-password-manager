package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type StoredCredentials struct {
	Token        string `json:"token"`
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Salt         string `json:"salt"`          // соль для Argon2 (base64)
	EncryptedKey string `json:"encrypted_key"` // зашифрованный ключ (опционально)
}

type FileStorage struct {
	path string
}

func NewFileStorage() (*FileStorage, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(home, ".gophkeeper")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}

	return &FileStorage{
		path: filepath.Join(dir, "session.json"),
	}, nil
}

func (fs *FileStorage) Save(creds *StoredCredentials) error {
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.path, data, 0600)
}

func (fs *FileStorage) Load() (*StoredCredentials, error) {
	data, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var creds StoredCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	return &creds, nil
}

func (fs *FileStorage) Clear() error {
	if err := os.Remove(fs.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
