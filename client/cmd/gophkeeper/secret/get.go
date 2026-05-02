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

// SecretGetCmd - получение и расшифровка секрета
var SecretGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get and decrypt a secret by ID",
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

		if len(app.App.Salt) == 0 {
			fmt.Println("Error: No salt in session. Login again.")
			return
		}

		// Мастер-пароль
		fmt.Print("Enter master password: ")
		masterPass, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		masterPass = strings.TrimSpace(masterPass)

		if masterPass == "" {
			fmt.Println(domain.ErrMasterPasswordRequired)
			return
		}

		// Получаем секрет с сервера
		secret, err := app.App.RestClient.GetSecret(id)
		if err != nil {
			fmt.Printf("Error fetching secret: %v\n", err)
			return
		}

		// Извлекаем зашифрованные данные (приходит как строка)
		encryptedStr, ok := secret["data"].(string)
		if !ok {
			fmt.Println("Error: no data field")
			return
		}

		// Декодируем base64 в []byte
		encryptedData, err := base64.StdEncoding.DecodeString(encryptedStr)
		if err != nil {
			fmt.Printf("Base64 decode error: %v\n", err)
			return
		}

		// Генерируем ключ
		key, err := app.App.KeyManager.DeriveKeyFromMasterPassword(masterPass, app.App.Salt)
		if err != nil {
			fmt.Printf("Key error: %v\n", err)
			return
		}

		// Расшифровываем
		decrypted, err := app.App.KeyManager.Decrypt(encryptedData, key)
		if err != nil {
			fmt.Printf("Decryption failed: %v\n", err)
			return
		}

		fmt.Printf("Decrypted data: %s\n", string(decrypted))

		// Выводим результат
		fmt.Println("\n Secret Detail")
		fmt.Printf("Title: %v\n", secret["title"])
		fmt.Printf("Type: %v\n", secret["type"])
		fmt.Printf("Data: %s\n", string(decrypted))
		fmt.Printf("Created: %v\n", secret["created_at"])
	},
}

func init() {
	SecretGetCmd.Flags().StringP("id", "i", "", "Secret ID")
	SecretGetCmd.MarkFlagRequired("id")
}
