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

// SecretDeleteCmd — команда для удаления секрета по ID.
//
// Использование:
//
//	gophkeeper secret delete --id <secret-id>
//	gophkeeper secret delete -i <secret-id>
//
// Флаги:
//   - --id, -i: идентификатор секрета для удаления (обязательный)
//
// Алгоритм:
//  1. Проверяет наличие ID секрета
//  2. Проверяет аутентификацию пользователя
//  3. Отправляет запрос на удаление на сервер
//  4. Выводит результат операции
var SecretDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a secret by ID",
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

		// Отправка запроса на удаление
		err := app.App.RestClient.DeleteSecret(id)
		if err != nil {
			fmt.Printf("Error deleting secret: %v\n", err)
			return
		}

		// Успешный ответ
		fmt.Printf("Secret with ID %s successfully deleted.\n", id)
	},
}

// init регистрирует флаги команды SecretDeleteCmd.
func init() {
	// Определение флага ID
	SecretDeleteCmd.Flags().StringP("id", "i", "", "Secret ID to delete")
	// Обязательный флаг
	SecretDeleteCmd.MarkFlagRequired("id")
}
