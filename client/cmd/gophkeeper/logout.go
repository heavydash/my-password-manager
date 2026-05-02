package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
)

// logoutCmd
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear saved session",
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.App.Storage.Clear(); err != nil {
			fmt.Printf("Error clearing session: %v\n", err)
			return
		}
		app.App.Token = ""
		app.App.UserID = ""
		app.App.Email = ""
		app.App.Salt = nil
		app.App.RestClient.SetToken("")
		fmt.Println("Logged out. Session cleared.")
	},
}

func init() {
	RootCmd.AddCommand(logoutCmd)
}
