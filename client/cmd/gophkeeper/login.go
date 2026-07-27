// Package main содержит команды CLI для GophKeeper client.
//
// Содержит команды:
//   - login: аутентификация на сервере
//   - register: регистрация нового пользователя
//   - secret: управление секретами (add, get, list, delete)
package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/adapters/encryption"
	"gophkeeper/client/internal/adapters/storage"
	"gophkeeper/client/internal/app"
)

// loginCmd — команда для аутентификации пользователя на сервере.
//
// Использование:
//
//	gophkeeper login --email user@example.com --password mypass
//	gophkeeper login -e user@example.com -p mypass
//
// Флаги:
//   - --email, -e: email пользователя (обязательный)
//   - --password, -p: пароль пользователя (обязательный)
//
// Алгоритм:
//  1. Проверяет наличие email и пароля
//  2. Отправляет запрос на сервер для аутентификации
//  3. Генерирует соль (если её нет)
//  4. Деривирует ключ из мастер-пароля
//  5. Сохраняет сессию (токен, userID, email, соль) в локальное хранилище
//  6. Устанавливает токен для HTTP клиента
//
// Примечание:
//   - Соль генерируется при первом входе и сохраняется локально
//   - При последующих входах соль восстанавливается из сохранённой сессии
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GophKeeper",
	Run: func(cmd *cobra.Command, args []string) {
		// Извлечение флагов
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		// Валидация входных данных
		if email == "" || password == "" {
			fmt.Println("Error: --email and --password are required")
			return
		}

		// Отправка запроса на сервер
		resp, err := app.App.RestClient.Login(map[string]string{
			"email": email, "password": password,
		})
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		// Генерация соли (только при первом входе)
		if len(app.App.Salt) == 0 {
			salt, err := app.App.KeyManager.GenerateSalt()
			if err != nil {
				fmt.Printf("Error generating salt: %v\n", err)
				return
			}
			app.App.Salt = salt
		}

		// Деривация ключа для проверки (не сохраняется)
		key, err := app.App.KeyManager.DeriveKeyFromMasterPassword(password, app.App.Salt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Key length: %d bytes\n", len(key))

		// Сохранение сессии в локальное хранилище
		err = app.App.Storage.Save(&storage.StoredCredentials{
			Token:  resp.Token,
			UserID: resp.UserID,
			Email:  email,
			Salt:   encryption.SaltToString(app.App.Salt),
		})
		if err != nil {
			fmt.Printf("Warning: couldn't save session: %v\n", err)
			fmt.Println("You are still logged in, but session won't persist after restart")
		} else {
			fmt.Println("Session saved successfully")
		}

		// Обновление состояния приложения
		app.App.Token = resp.Token
		app.App.UserID = resp.UserID
		app.App.Email = email
		app.App.RestClient.SetToken(resp.Token)

		fmt.Println("Login successful! Session saved.")
	},
}

// init регистрирует флаги команды loginCmd и добавляет её в root команду.
func init() {
	// Определение флагов
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	loginCmd.Flags().StringP("password", "p", "", "Password")

	// Обязательные флаги
	loginCmd.MarkFlagRequired("email")
	loginCmd.MarkFlagRequired("password")

	// Добавление команды в root
	RootCmd.AddCommand(loginCmd)
}
