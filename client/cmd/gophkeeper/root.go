// Package main содержит root команду CLI клиента GophKeeper.
package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/cmd/gophkeeper/secret"
	"os"

	_ "gophkeeper/client/cmd/gophkeeper/secret"
)

// RootCmd — корневая команда CLI.
//
// Использование:
//
//	gophkeeper [command]
//
// Доступные команды:
//   - login      Вход в систему
//   - logout     Выход из системы
//   - register   Регистрация нового пользователя
//   - secret     Управление секретами
//
// Примеры:
//
//	gophkeeper --help
//	gophkeeper login --email user@example.com --password mypass
//	gophkeeper secret add --title "My Password" --type password
//	gophkeeper secret list
//	gophkeeper secret get --id <secret-id>
//	gophkeeper secret delete --id <secret-id>
//	gophkeeper logout
var RootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper - secure password and secret manager",
	Long:  "GophKeeper is a secure, end-to-end encrypted password manager.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("GophKeeper CLI")
		fmt.Println("Use --help to see available commands")
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Создаём группу secret и добавляем команды
	secretCmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage secrets",
	}

	secretCmd.AddCommand(secret.SecretAddCmd, secret.SecretListCmd, secret.SecretGetCmd, secret.SecretDeleteCmd)

	RootCmd.AddCommand(secretCmd)
}
