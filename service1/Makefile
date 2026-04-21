.PHONY: test test-unit test-integration test-coverage test-verbose

# Запуск всех тестов
test:
	go test ./... -v

# Запуск только unit тестов
test-unit:
	go test ./... -v -short

# Запуск интеграционных тестов
test-integration:
	go test ./... -v -tags=integration

# Запуск с покрытием
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Запуск с детальным выводом
test-verbose:
	go test ./... -v -race -cover

# Запуск конкретного теста
test-single:
	go test ./... -v -run $(TEST_NAME)

# Очистка кеша тестов
test-clean:
	go clean -testcache