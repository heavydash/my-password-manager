package secret

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
)

// SecretDeleteCmd — удаление секрета
var SecretDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a secret by ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Println("Error: --id is required")
			return
		}

		if app.App.Token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		err := app.App.RestClient.DeleteSecret(id)
		if err != nil {
			fmt.Printf("Error deleting secret: %v\n", err)
			return
		}

		fmt.Printf("Secret with ID %s successfully deleted.\n", id)
	},
}

func init() {
	SecretDeleteCmd.Flags().StringP("id", "i", "", "Secret ID to delete")
	SecretDeleteCmd.MarkFlagRequired("id")
}
