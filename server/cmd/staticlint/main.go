// Package main реализует мультичекер для статического анализа кода GophKeeper.
//
// Собирает и запускает набор анализаторов:
//   - Стандартные анализаторы из x/tools (assign, shadow, printf, nilness и др.)
//   - Анализаторы staticcheck (SAxxxx, ST1000)
//   - Кастомный анализатор hardcodechecker
//   - Публичные анализаторы errcheck и nilerr
//
// Использование:
//
//	go run ./cmd/linter -h
//	go run ./cmd/linter ./...
package main

import (
	"github.com/gostaticanalysis/nilerr"
	"github.com/kisielk/errcheck/errcheck"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"gophkeeper/server/cmd/staticlint/hardcodechecker"
	"honnef.co/go/tools/staticcheck"
	"strings"
)

// Multichecker для GophKeeper
func main() {
	// Инициализация слайса со стандартными анализаторами
	// Все анализаторы из пакета passes гарантированно работают без дополнительных настроек
	analyzers := []*analysis.Analyzer{
		assign.Analyzer,      // Поиск ненужных присваиваний
		shadow.Analyzer,      // Поиск переменных, перекрывающих внешние
		printf.Analyzer,      // Проверка форматирования в fmt.Printf и подобных
		nilness.Analyzer,     // Анализ nil-значений и потенциальных паник
		lostcancel.Analyzer,  // Поиск отменённых контекстов без вызова cancel
		copylock.Analyzer,    // Поиск копирования sync.Mutex и подобных
		loopclosure.Analyzer, // Поиск проблем с замыканиями в цикле
	}

	// Добавление анализаторов staticcheck
	for _, sa := range staticcheck.Analyzers {
		// SAxxxx — основные проверки безопасности и ошибок
		if strings.HasPrefix(sa.Analyzer.Name, "SA") {
			analyzers = append(analyzers, sa.Analyzer)
		}
	}

	// ST1000 — проверка наличия комментария к пакету
	for _, sa := range staticcheck.Analyzers {
		if sa.Analyzer.Name == "ST1000" {
			analyzers = append(analyzers, sa.Analyzer)
			break
		}
	}

	// Добавление кастомных и публичных анализаторов
	// hardcodechecker — собственный анализатор для поиска хардкод секретов
	// errcheck — проверка обработки ошибок
	// nilerr — поиск ошибок, которые всегда nil
	analyzers = append(analyzers, hardcodechecker.Analyzer)
	analyzers = append(analyzers, errcheck.Analyzer)
	analyzers = append(analyzers, nilerr.Analyzer)

	// Запуск мультичекера с собранным набором анализаторов
	multichecker.Main(analyzers...)
}
