package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
)

// registerCmd
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

func init() {
	registerCmd.Flags().StringP("email", "e", "", "Email address")
	registerCmd.Flags().StringP("password", "p", "", "Password")
	registerCmd.MarkFlagRequired("email")
	registerCmd.MarkFlagRequired("password")

	RootCmd.AddCommand(registerCmd)
}
