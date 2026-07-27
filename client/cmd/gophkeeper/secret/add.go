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

// SecretAddCmd — команда для добавления нового секрета.
//
// Использование:
//
//	gophkeeper secret add --title "My Password" --type password
//
// Флаги:
//   - --title, -t: заголовок секрета (обязательный)
//   - --type, -y: тип секрета (password, note, card, ssh_key, custom)
//
// Алгоритм:
//  1. Проверяет аутентификацию пользователя
//  2. Запрашивает мастер-пароль
//  3. Запрашивает данные секрета
//  4. Шифрует данные мастер-паролем
//  5. Отправляет зашифрованные данные на сервер
var SecretAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new secret",
	Run: func(cmd *cobra.Command, args []string) {
		// Извлечение флагов командной строки
		title, _ := cmd.Flags().GetString("title")
		secretType, _ := cmd.Flags().GetString("type")

		// Валидация заголовка
		if title == "" {
			fmt.Println("Error: --title is required")
			return
		}

		// Проверка аутентификации
		if app.App.Token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		// Установка токена для HTTP клиента
		app.App.RestClient.SetToken(app.App.Token)

		// Проверка целостности сессии
		if len(app.App.Salt) == 0 {
			fmt.Println("Error: Session corrupted. Please login again")
		}

		// Запрос мастер-пароля у пользователя
		fmt.Print("Enter master password: ")
		masterPass, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		masterPass = strings.TrimSpace(masterPass)

		if masterPass == "" {
			fmt.Println(domain.ErrMasterPasswordRequired)
			return
		}

		// Запрос данных секрета
		fmt.Print("Enter secret data (login:password or note text): ")
		data, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		data = strings.TrimSpace(data)

		if data == "" {
			fmt.Println(domain.ErrInvalidInput)
			return
		}

		// Деривация ключа шифрования из мастер-пароля
		key, err := app.App.KeyManager.DeriveKeyFromMasterPassword(masterPass, app.App.Salt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Шифрование данных
		encrypted, err := app.App.KeyManager.Encrypt([]byte(data), key)
		if err != nil {
			fmt.Printf("Encryption failed: %v\n", err)
			return
		}

		// Кодирование в base64 для передачи по HTTP
		encryptedBase64 := base64.StdEncoding.EncodeToString(encrypted)

		// Отправка на сервер
		secretID, err := app.App.RestClient.CreateSecret(title, secretType, string(encryptedBase64))
		if err != nil {
			fmt.Printf("Server error: %v\n", err)
			return
		}

		// Успешный ответ
		fmt.Printf(" Secret added successfully!\n")
		fmt.Printf("   ID: %s\n", secretID)
		fmt.Printf("   Title: %s\n", title)
	},
}

// init регистрирует флаги команды SecretAddCmd.
func init() {
	// Определение флагов
	SecretAddCmd.Flags().StringP("title", "t", "", "Title of the secret")
	SecretAddCmd.Flags().StringP("type", "y", "password", "Type of secret")
	// Обязательные флаги
	SecretAddCmd.MarkFlagRequired("title")
}
