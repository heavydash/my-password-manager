// Package main — точка входа для CLI клиента GophKeeper.
//
// Приложение позволяет:
//   - Регистрироваться и входить в систему
//   - Управлять секретами (добавление, просмотр, удаление)
//   - Хранить зашифрованные данные на сервере
//
// Использование:
//
//	gophkeeper --help
//	gophkeeper login --email user@example.com --password mypass
//	gophkeeper secret add --title "My Password" --type password
//	gophkeeper secret list
//	gophkeeper secret get --id <secret-id>
//	gophkeeper secret delete --id <secret-id>
//	gophkeeper logout
package main

import (
	_ "gophkeeper/client/cmd/gophkeeper/secret"
	"gophkeeper/client/internal/app"
)

// main — точка входа в приложение.
//
// Алгоритм:
//  1. Инициализирует глобальное состояние приложения (app.App)
//  2. Загружает сохранённую сессию (если есть)
//  3. Выполняет парсинг и обработку команд CLI
func main() {
	// Инициализация приложения:
	//   - Загрузка конфигурации
	//   - Инициализация HTTP клиента
	//   - Инициализация KeyManager
	//   - Загрузка сохранённой сессии
	app.InitApp()

	// Запуск обработки команд CLI (cobra)
	Execute()
}
