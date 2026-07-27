// Package secret реализует команды CLI для управления секретами.
//
// Содержит команды:
//   - add: добавление нового секрета
//   - get: получение секрета
//   - list: список секретов
//   - delete: удаление секрета
package secret

import (
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/app"
)

// SecretListCmd — команда для получения списка всех секретов пользователя.
//
// Использование:
//
//	gophkeeper secret list
//
// Алгоритм:
//  1. Проверяет аутентификацию пользователя
//  2. Запрашивает список секретов с сервера
//  3. Выводит список в удобочитаемом формате
//  4. Если секретов нет, сообщает об этом
var SecretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all your secrets",
	Long:  "List all your secrets",
	Run: func(cmd *cobra.Command, args []string) {
		// Проверка аутентификации
		if app.App.Token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		// Установка токена для HTTP клиента
		app.App.RestClient.SetToken(app.App.Token)

		fmt.Println("Fetching your secrets from server...")

		// Получение списка секретов с сервера
		secrets, err := app.App.RestClient.GetSecrets()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// Проверка наличия секретов
		if len(secrets) == 0 {
			fmt.Println("No secrets found.")
			return
		}

		// Вывод списка секретов
		fmt.Println("\n Your secrets: ")
		for i, s := range secrets {
			title := s["title"]
			secretType := s["type"]
			createdAt := s["created_at"]
			fmt.Printf("%d. %s (type: %s) created: %v\n", i+1, title, secretType, createdAt)
		}

	},
}

// init регистрирует флаги команды SecretListCmd (если потребуются).
func init() {
	// Флаги для команды list (пока не требуются)

}
