package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/cmd/gophkeeper/secret"
	"os"

	_ "gophkeeper/client/cmd/gophkeeper/secret"
)

// RootCmd
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
