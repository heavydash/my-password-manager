package secret

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
	"gophkeeper/client/internal/domain"
	"os"
	"strings"
)

// SecretAddCmd
var SecretAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new secret",
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		secretType, _ := cmd.Flags().GetString("type")

		if title == "" {
			fmt.Println("Error: --title is required")
			return
		}

		if app.App.Token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		// Принудительно устанавливаем токен
		app.App.RestClient.SetToken(app.App.Token)

		if len(app.App.Salt) == 0 {
			fmt.Println("Error: Session corrupted. Please login again")
		}

		// Мастер-пароль
		fmt.Print("Enter master password: ")
		masterPass, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		masterPass = strings.TrimSpace(masterPass)

		if masterPass == "" {
			fmt.Println(domain.ErrMasterPasswordRequired)
			return
		}

		// Данные секрета
		fmt.Print("Enter secret data (login:password or note text): ")
		data, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		data = strings.TrimSpace(data)

		if data == "" {
			fmt.Println(domain.ErrInvalidInput)
			return
		}

		// Шифрование
		key, err := app.App.KeyManager.DeriveKeyFromMasterPassword(masterPass, app.App.Salt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		encrypted, err := app.App.KeyManager.Encrypt([]byte(data), key)
		if err != nil {
			fmt.Printf("Encryption failed: %v\n", err)
			return
		}

		encryptedBase64 := base64.StdEncoding.EncodeToString(encrypted)

		secretID, err := app.App.RestClient.CreateSecret(title, secretType, string(encryptedBase64))
		if err != nil {
			fmt.Printf("Server error: %v\n", err)
			return
		}

		fmt.Printf(" Secret added successfully!\n")
		fmt.Printf("   ID: %s\n", secretID)
		fmt.Printf("   Title: %s\n", title)
	},
}

func init() {
	SecretAddCmd.Flags().StringP("title", "t", "", "Title of the secret")
	SecretAddCmd.Flags().StringP("type", "y", "password", "Type of secret")
	SecretAddCmd.MarkFlagRequired("title")
}
