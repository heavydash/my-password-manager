package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/adapters/encryption"
	"gophkeeper/client/internal/adapters/storage"
	"gophkeeper/client/internal/app"
)

// loginCmd
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GophKeeper",
	Run: func(cmd *cobra.Command, args []string) {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			fmt.Println("Error: --email and --password are required")
			return
		}

		resp, err := app.App.RestClient.Login(map[string]string{
			"email": email, "password": password,
		})
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		if len(app.App.Salt) == 0 {
			salt, err := app.App.KeyManager.GenerateSalt()
			if err != nil {
				fmt.Printf("Error generating salt: %v\n", err)
				return
			}
			app.App.Salt = salt
		}

		// Выводим ключ
		key, err := app.App.KeyManager.DeriveKeyFromMasterPassword(password, app.App.Salt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Key length: %d bytes\n", len(key))

		// Сохраняем токен и метаданные
		err = app.App.Storage.Save(&storage.StoredCredentials{
			Token:  resp.Token,
			UserID: resp.UserID,
			Email:  email,
			Salt:   encryption.SaltToString(app.App.Salt),
		})
		if err != nil {
			fmt.Printf("Warning: couldn't save session: %v\n", err)
		}

		app.App.Token = resp.Token
		app.App.UserID = resp.UserID
		app.App.Email = email
		app.App.RestClient.SetToken(resp.Token)

		fmt.Println("Login successful! Session saved.")
	},
}

func init() {
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	loginCmd.Flags().StringP("password", "p", "", "Password")
	loginCmd.MarkFlagRequired("email")
	loginCmd.MarkFlagRequired("password")

	RootCmd.AddCommand(loginCmd)
}
