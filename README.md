# Словарно-тренировочный сервис

CLI-приложение для изучения иностранных слов с поддержкой китайского, русского и английского языков.

## Архитектура

```
┌─────────────────┐         ┌─────────────────┐         ┌────────────────┐
│   CLI Клиент    │ ──────► │   Сервис-1      │ ──────► │   Сервис-2     │
│   (терминал)    │  HTTP   │   :8081         │  HTTP   │    :8083       │
│                 │ ◄────── │                 │ ◄────── │                │
└─────────────────┘         └─────────────────┘         └───────┬────────┘
                                                                │
                                                                ▼
                                                            ┌─────────────┐
                                                            │ PostgreSQL  │
                                                            └─────────────┘
```

<img width="725" height="668" alt="image" src="https://github.com/user-attachments/assets/d9924f73-0b3d-406e-89cd-6542912402e5" />



- **Сервис-1 (API Gateway)** — HTTP-сервер, принимает запросы от CLI, валидирует, обращается к Сервису-2
- **Сервис-2 (Dictionary Engine)** — HTTP-сервер, бизнес-логика, работа с БД
- **CLI Клиент** — отдельная программа, отправляет HTTP-запросы к Сервису-1

## Быстрый старт

### 1. Запуск Сервиса-2 (словарь + БД)

```bash
go run cmd/service2/main.go
```

Ожидаемый вывод:
```
2026/03/20 10:00:00 Сервис-2 запущен на :8083
2026/03/20 10:00:00 Подключение к PostgreSQL установлено
```

### 2. Запуск Сервиса-1 (API Gateway)

```bash
go run cmd/service1/main.go
```

Ожидаемый вывод:
```
2026/03/20 10:00:05 Сервис-1 запущен на :8081
2026/03/20 10:00:05 Подключение к Сервису-2: http://localhost:8083
```

### 3. Использование CLI Клиента

```bash
# Перевод слова
go run cmd/cli/main.go --source zh --target ru --word "zha3o"

# Слова по теме на одном языке
go run cmd/cli/main.go --topic animals --language ru

# Слова по теме на нескольких языках
go run cmd/cli/main.go --topic animals --languages ru,en,zh

# Проверка перевода
go run cmd/cli/main.go --check --original "искать" --translation "zha3o" --lang ru

# Список языков
go run cmd/cli/main.go --list-languages

# Список тем
go run cmd/cli/main.go --list-topics
```

## Команды CLI

### Флаги

| Флаг | Описание | Пример |
|------|----------|--------|
| `--server` | Адрес Сервиса-1 | `--server http://localhost:8080` |
| `--source` | Язык оригинала | `--source zh` |
| `--target` | Целевой язык | `--target ru` |
| `--word` | Слово для перевода | `--word "zha3o"` |
| `--topic` | Тема для получения слов | `--topic animals` |
| `--language` | Один язык | `--language ru` |
| `--languages` | Несколько языков (через запятую) | `--languages ru,en,zh` |
| `--check` | Режим проверки | `--check` |
| `--original` | Исходное слово | `--original "искать"` |
| `--translation` | Перевод пользователя | `--translation "zha3o"` |
| `--lang` | Язык оригинала (для проверки) | `--lang ru` |
| `--format` | Формат вывода (table/json) | `--format json` |
| `--list-languages` | Список языков | `--list-languages` |
| `--list-topics` | Список тем | `--list-topics` |

### Примеры

```bash
# Указание сервера (по умолчанию http://localhost:8080)
go run cmd/cli/main.go --server http://localhost:8080 --source en --target ru --word "hello"

# Сравнение трех языков в табличном формате
go run cmd/cli/main.go --topic animals --languages ru,en,zh --format table

# Экспорт в JSON
go run cmd/cli/main.go --topic food --languages ru,en --format json

# Проверка перевода
go run cmd/cli/main.go --check --original "собака" --translation "dog" --lang ru
```


## API Endpoints (Сервис-1)

Сервис-1 принимает HTTP-запросы от CLI и проксирует их в Сервис-2.

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/v1/translate` | Перевод слова |
| POST | `/api/v1/topics/words` | Слова по теме (один язык) |
| POST | `/api/v1/topics/words-multi` | Слова по теме (несколько языков) |
| POST | `/api/v1/check-translation` | Проверка перевода |
| GET | `/api/v1/languages` | Список языков |
| GET | `/api/v1/topics` | Список тем |
| GET | `/api/v1/health` | Проверка здоровья |

## API Endpoints (Сервис-2)

Сервис-2 принимает JSON-запросы от Сервиса-1.

| Метод | Endpoint | Описание |
|-------|----------|----------|
| POST | `/api/v1/translate` | Перевод слова |
| POST | `/api/v1/topics/words` | Слова по теме (один язык) |
| POST | `/api/v1/topics/words-multi` | Слова по теме (несколько языков) |
| POST | `/api/v1/check-translation` | Проверка перевода |
| GET | `/api/v1/languages` | Список языков |
| GET | `/api/v1/topics` | Список тем |
| GET | `/api/v1/health` | Проверка здоровья |


## Конфигурация

### Сервис-1 (`configs/service1.json`)
```json
{
  "port": 8080,
  "log_level": "info",
  "service2_url": "http://localhost:8081",
  "timeout_seconds": 10
}
```

### Сервис-2 (`configs/service2.json`)
```json
{
  "port": 8081,
  "log_level": "info",
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "postgres",
    "password": "postgres",
    "dbname": "dictionary",
    "ssl_mode": "disable"
  }
}
```

### CLI (`configs/cli.json`)
```json
{
  "server_url": "http://localhost:8080",
  "timeout_seconds": 10,
  "default_format": "table"
}
```

## Запуск всех компонентов

### Вариант 1: Ручной запуск (3 терминала)

**Терминал 1 — Сервис-2:**
```bash
go run cmd/service2/main.go
```

**Терминал 2 — Сервис-1:**
```bash
go run cmd/service1/main.go
```

**Терминал 3 — CLI Клиент:**
```bash
go run cmd/cli/main.go --list-languages
```

### Вариант 2: Использование Makefile

```bash
# Запуск всех сервисов
make run

# Запуск только Сервиса-2
make run-service2

# Запуск только Сервиса-1
make run-service1

# Запуск CLI с примером
make cli-example
```

### Makefile
```makefile
run-service2:
	go run cmd/service2/main.go

run-service1:
	go run cmd/service1/main.go

run:
	go run cmd/service2/main.go &
	go run cmd/service1/main.go &

cli-example:
	go run cmd/cli/main.go --list-languages

test:
	go test ./... -v

.PHONY: run-service2 run-service1 run cli-example test
```

## Тестирование

```bash
# Все тесты
go test ./... -v

# Юнит-тесты с покрытием
go test ./tests/unit/... -cover

# Интеграционные тесты (требуют запущенные сервисы)
go test ./tests/integration/... -v
```

## Ошибки
### Список всех ошибок

- 400 Bad Request
Не поддерживается язык (Unsupported lang)
Некорректный формат ввода (Invalid JSON)
Нарушение ограничения полей (Validation failed)
Не заполнены обязательные поля (Missing params)
Неверные флаги

- 500 Internal Server Error
Упал сервис первый или второй (Service unavailable)
Нет ответа > 1 минуты (Timeout)
Исключение на втором сервисе (Internal error)

- ошибки БД
Connection failed — Postgres недоступен
Query failed — Ошибка SQL запроса/синтаксиса
Transaction failed — Deadlock/нарушение уникальности

- бизнес ошибки
TranslationNotFound — Слово не найдено в БД
RateLimitExceeded — Слишком много запросов

### Примеры шибок

#### CLI Клиент
```bash
# Сервис-1 не запущен
$ go run cmd/cli/main.go --list-languages
Ошибка: не удалось подключиться к серверу http://localhost:8080
Убедитесь, что Сервис-1 запущен

# Слово не найдено
$ go run cmd/cli/main.go --source zh --target ru --word "nonexistent"
Ошибка: слово не найдено в словаре
```

#### Сервис-1
При ошибке обращения к Сервису-2 возвращает:
```json
{
  "success": false,
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Сервис-2 недоступен"
  }
}
```

## Требования

- Go 1.21+
- PostgreSQL 14+
