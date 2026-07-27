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

// registerCmd — команда для регистрации нового пользователя.
//
// Использование:
//
//	gophkeeper register --email user@example.com --password mypass
//	gophkeeper register -e user@example.com -p mypass
//
// Флаги:
//   - --email, -e: email пользователя (обязательный)
//   - --password, -p: пароль пользователя (обязательный)
//
// Алгоритм:
//  1. Проверяет наличие email и пароля
//  2. Отправляет запрос на сервер для регистрации
//  3. Выводит результат регистрации
//
// Примечание:
//   - Пароль должен быть надёжным (мин. 8 символов)
//   - После регистрации нужно выполнить вход командой 'login'
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register new user",
	Run: func(cmd *cobra.Command, args []string) {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			fmt.Println("Error: --email and --password are required")
			cmd.Usage()
			return
		}

		fmt.Printf("Registering user with email: %s\n", email)

		err := app.App.RestClient.Register(email, password)
		if err != nil {
			fmt.Printf("Registration failed: %v\n", err)
			return
		}

		fmt.Println("Registration successful. You can now login!")
	},
}

// init регистрирует флаги команды registerCmd и добавляет её в root команду.
func init() {
	// Определение флагов
	registerCmd.Flags().StringP("email", "e", "", "Email address")
	registerCmd.Flags().StringP("password", "p", "", "Password")
	// Обязательные флаги
	registerCmd.MarkFlagRequired("email")
	registerCmd.MarkFlagRequired("password")

	// Добавление команды в root
	RootCmd.AddCommand(registerCmd)
}
