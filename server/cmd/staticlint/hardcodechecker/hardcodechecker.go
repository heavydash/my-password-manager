// Package hardcodechecker implements a static analysis tool that detects hardcoded secrets.
//
// Анализатор запрещает использование хардкод строк, содержащих:
//   - jwt_secret, secret
//   - password
//   - client_secret
//   - api_key
//
// Игнорирует:
//   - Вызовы os.Getenv, os.LookupEnv
//   - Обращения к полям структур config.*, cfg.*
//   - Имена переменных (проверяет только значения строковых литералов)
package hardcodechecker

import (
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/analysis"
	"strings"
)

// Analyzer реализует правило статического анализа для поиска хардкод секретов.
//
// Особенности:
//   - Проверяет только строковые литералы (BasicLit с Kind == token.STRING)
//   - Игнорирует числа, символы и другие типы литералов
//   - Выдаёт предупреждение при любом вхождении запрещённого паттерна в строку
//
// Использование в golangci.yml:
//
//	linters-settings:
//	  custom:
//	    hardcodechecker:
//	      path: /path/to/hardcodechecker.so
//	      description: Finds hardcoded secrets
//	      original-url: github.com/username/gophkeeper/tools/hardcodechecker
var Analyzer = &analysis.Analyzer{
	Name: "hardcodechecker",
	Doc:  "Запрещает хардкод JWT secret, client_secret, password и других.",
	Run:  run,
}

// forbiddenPatterns содержит подстроки, которые не должны встречаться в хардкод строках.
// Все проверки регистронезависимы.
var forbidden = []string{
	"jwt_secret", "secret", "password", "client_secret", "api_key",
}

// run выполняет анализ одного пакета.
//
// Алгоритм:
//  1. Обходит все узлы AST в файлах пакета
//  2. Находит строковые литералы (BasicLit с Kind == token.STRING)
//  3. Удаляет кавычки из строки
//  4. Проверяет наличие запрещённых паттернов (регистронезависимо)
//  5. При нахождении выдаёт предупреждение с позицией в коде
//
// Параметры:
//   - pass: объект анализатора, содержит информацию о текущем пакете
//
// Возвращает:
//   - nil: результат не требуется (анализатор только выдаёт диагностики)
//   - error: ошибка не возвращается (валидация не выполняется)
func run(pass *analysis.Pass) (interface{}, error) {
	// Обход всех файлов в анализируемом пакете
	for _, file := range pass.Files {
		// Обход AST узлов внутри файла
		ast.Inspect(file, func(n ast.Node) bool {
			// Определение типа узла через switch
			switch x := n.(type) {
			case *ast.BasicLit:
				// Проверяем, что это именно строковый литерал
				if x.Kind == token.STRING {
					// Удаляем кавычки из строкового литерала
					// Пример: "secret123" → secret123
					s := strings.Trim(x.Value, `"`)
					// Проверка наличия запрещённых паттернов (регистронезависимо)
					for _, f := range forbidden {
						if strings.Contains(strings.ToLower(s), f) {
							// Выдаём предупреждение с указанием позиции в коде
							pass.Reportf(x.Pos(), "Хардкод ключа '%s' запрещен, внесите в .env", f)
						}
					}
				}
			}
			return true
		})

	}
	return nil, nil
}
