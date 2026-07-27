// Package main содержит команды CLI для GophKeeper client.
//
// Содержит команды:
//   - login: аутентификация на сервере
//   - logout: завершение сессии
//   - register: регистрация нового пользователя
//   - secret: управление секретами (add, get, list, delete)
package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
)

// logoutCmd — команда для завершения сессии и очистки локальных данных.
//
// Использование:
//
//	gophkeeper logout
//
// Алгоритм:
//  1. Очищает локальное хранилище сессии (токен, userID, email, соль)
//  2. Сбрасывает состояние приложения
//  3. Очищает токен в HTTP клиенте
//
// Примечание:
//   - Команда не требует подключения к серверу
//   - Все локальные данные аутентификации удаляются
//   - Секреты остаются на сервере (только сессия очищается)
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear saved session",
	Run: func(cmd *cobra.Command, args []string) {
		// Проверка наличия активной сессии
		if app.App.Token == "" {
			fmt.Println("You are not logged in.")
			return
		}

		// Очистка локального хранилища
		if err := app.App.Storage.Clear(); err != nil {
			fmt.Printf("Warning: failed to clear session storage: %v\n", err)
			fmt.Println("Session data will still be cleared from memory")
		} else {
			fmt.Println("Session storage cleared successfully")
		}

		// Сброс состояния приложения
		app.App.Token = ""
		app.App.UserID = ""
		app.App.Email = ""
		app.App.Salt = nil
		app.App.RestClient.SetToken("")

		fmt.Println("Logged out. Session cleared.")
	},
}

// init регистрирует команду logout в root команде.
func init() {
	RootCmd.AddCommand(logoutCmd)
}
