// Package secret реализует команды CLI для управления секретами.
//
// Содержит команды:
//   - add: добавление нового секрета
//   - get: получение секрета
//   - list: список секретов
//   - delete: удаление секрета
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

// SecretGetCmd — команда для получения и расшифровки секрета по ID.
//
// Использование:
//
//	gophkeeper secret get --id <secret-id>
//	gophkeeper secret get -i <secret-id>
//
// Флаги:
//   - --id, -i: идентификатор секрета (обязательный)
//
// Алгоритм:
//  1. Проверяет наличие ID и аутентификацию
//  2. Запрашивает мастер-пароль
//  3. Получает зашифрованный секрет с сервера
//  4. Деривирует ключ из мастер-пароля
//  5. Расшифровывает данные
//  6. Выводит результат
var SecretGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get and decrypt a secret by ID",
	Run: func(cmd *cobra.Command, args []string) {
		// Извлечение флага ID
		id, _ := cmd.Flags().GetString("id")

		// Валидация ID
		if id == "" {
			fmt.Println("Error: --id is required")
			return
		}

		// Проверка аутентификации
		if app.App.Token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		// Установка токена для HTTP клиента
		app.App.RestClient.SetToken(app.App.Token)

		// Проверка наличия соли в сессии
		if len(app.App.Salt) == 0 {
			fmt.Println("Error: No salt in session. Login again.")
			return
		}

		// Запрос мастер-пароля у пользователя
		fmt.Print("Enter master password: ")
		masterPass, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		masterPass = strings.TrimSpace(masterPass)

		if masterPass == "" {
			fmt.Println(domain.ErrMasterPasswordRequired)
			return
		}

		// Получение зашифрованного секрета с сервера
		secret, err := app.App.RestClient.GetSecret(id)
		if err != nil {
			fmt.Printf("Error fetching secret: %v\n", err)
			return
		}

		// Извлечение зашифрованных данных из ответа
		encryptedStr, ok := secret["data"].(string)
		if !ok {
			fmt.Println("Error: no data field")
			return
		}

		// Декодирование base64 в байты
		encryptedData, err := base64.StdEncoding.DecodeString(encryptedStr)
		if err != nil {
			fmt.Printf("Base64 decode error: %v\n", err)
			return
		}

		// Деривация ключа из мастер-пароля
		key, err := app.App.KeyManager.DeriveKeyFromMasterPassword(masterPass, app.App.Salt)
		if err != nil {
			fmt.Printf("Key error: %v\n", err)
			return
		}

		// Расшифровка данных
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

// init регистрирует флаги команды SecretGetCmd.
func init() {
	// Определение флага ID
	SecretGetCmd.Flags().StringP("id", "i", "", "Secret ID")

	// Обязательный флаг
	SecretGetCmd.MarkFlagRequired("id")
}
