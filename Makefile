.PHONY: help check test coverage build-server pprof clean docker-up docker-down

help: ## Показать все команды
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## Запуск мультичекера
	cd server && go run ./cmd/linter || true

test: ## Запуск тестов
	go test ./server/... ./client/... -race -count=1 -timeout=30s

coverage: ## Покрытие тестов
	go test ./server/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

build-server: ## Сборка сервера
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../bin/server ./cmd/server

pprof: ## Запуск сервера с pprof (основная команда для разработки)
	go run ./server/cmd/server

docker-up: ## Запуск PostgreSQL
	docker compose up -d db

docker-down: ## Остановка PostgreSQL
	docker compose down

clean: ## Очистка
	rm -rf bin/ coverage.*