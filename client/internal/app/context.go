package app

import (
	"fmt"
	"gophkeeper/client/internal/adapters/encryption"
	"gophkeeper/client/internal/adapters/rest"
	"gophkeeper/client/internal/adapters/storage"
)

type Context struct {
	Token      string
	UserID     string
	Email      string
	Salt       []byte
	Storage    *storage.FileStorage
	KeyManager *encryption.KeyManager
	RestClient *rest.Client
}

// App Глобальный экземпляр
var App *Context

// InitApp — инициализация
func InitApp() {
	fileStorage, _ := storage.NewFileStorage()

	App = &Context{
		Storage:    fileStorage,
		KeyManager: encryption.NewKeyManager(),
		RestClient: rest.NewClient("http://localhost:8080"),
	}

	// Загрузка сессии
	if fileStorage != nil {
		if saved, err := fileStorage.Load(); err == nil && saved != nil && saved.Token != "" {
			App.Token = saved.Token
			App.UserID = saved.UserID
			App.Email = saved.Email
			App.RestClient.SetToken(saved.Token)

			if saved.Salt != "" {
				if salt, err := encryption.StringToSalt(saved.Salt); err == nil {
					App.Salt = salt
				}
			}
			fmt.Printf("Welcome back, %s!\n", App.Email)
		}
	}
}
