package secret

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
)

// SecretListCmd
var SecretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all your secrets",
	Long:  "List all your secrets",
	Run: func(cmd *cobra.Command, args []string) {
		if app.App.Token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}
		fmt.Println("Fetching your secrets from server...")

		secrets, err := app.App.RestClient.GetSecrets()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if len(secrets) == 0 {
			fmt.Println("No secrets found.")
			return
		}

		fmt.Println("\n Your secrets: ")
		for i, s := range secrets {
			title := s["title"]
			secretType := s["type"]
			createdAt := s["created_at"]
			fmt.Printf("%d. %s (type: %s) created: %v\n", i+1, title, secretType, createdAt)
		}

	},
}

func init() {

}
