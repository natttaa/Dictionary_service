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
<img width="605" height="388" alt="image" src="https://github.com/user-attachments/assets/6ae48240-c92a-4b0b-98d5-1b68983e7fe9" />




- **Сервис-1 (API Gateway)** — HTTP-сервер, принимает запросы от CLI, валидирует, обращается к Сервису-2
- **Сервис-2 (Dictionary Engine)** — HTTP-сервер, бизнес-логика, работа с БД
- **CLI Клиент** — отдельная программа, отправляет HTTP-запросы к Сервису-1

## Быстрый старт

Команды ниже выполняются в PowerShell из корня репозитория `Dictionary_service`. Для работы нужны Go 1.24+ и доступ к PostgreSQL из `service2/config/service2.json`. Сервисы запускаются из **разных каталогов**, поскольку у каждого свой `go.mod`.

Перед запуском проверьте в `service2/config/service2.json` параметры `database.host`, `database.port`, `database.user`, `database.password`, `database.dbname` и `database.ssl_mode`. Не публикуйте пароль в README или коммитах. Адрес БД должен быть доступен с компьютера, где работает Сервис-2; для внутреннего адреса может понадобиться VPN.

### 1. Сервис-2 (терминал 1)

```powershell
cd .\service2
go run .
```

Оставьте терминал открытым. Дождитесь сообщений `Подключение к PostgreSQL успешно установлено` и `Запуск сервера` с адресом `:8083`. Если подключение к БД не удалось, сначала проверьте её адрес, учётные данные, доступ по сети и VPN.

### 2. Сервис-1 (терминал 2)

Откройте новый терминал **из корня `Dictionary_service`**:

```powershell
cd .\service1
go run cmd/service1/main.go
```

Оставьте и этот терминал открытым. В журнале должно появиться `Запуск сервера` с адресом `:8081`.

### 3. Проверка через CLI (терминал 3)

Откройте третий терминал **из корня `Dictionary_service`**:

```powershell
cd .\service1
go run cmd/cli/main.go --health
go run cmd/cli/main.go --list-languages
```

При успешном запуске проверка выводит `Статус сервиса: healthy` и `Словарный сервис: ok`. Если выводится `unhealthy`, проверьте журнал Сервиса-2 и соединение с PostgreSQL. Для остановки серверов нажмите `Ctrl+C` в первых двух терминалах.

## Мониторинг в Grafana

Оба сервиса отдают метрики запросов по адресу `/metrics`. Prometheus собирает их каждые 10 секунд, а Grafana показывает доступность сервисов, число запросов и ошибок по HTTP-кодам, а также среднее время ответа. Docker Compose запускает **только Prometheus и Grafana**; сами Go-сервисы по-прежнему запускаются на компьютере командами из раздела «Быстрый старт».

1. Запустите Docker Desktop и оба Go-сервиса на том же компьютере. Для Сервиса-2 нужно работающее соединение с PostgreSQL.
2. Из корня `Dictionary_service` запустите мониторинг:

   ```powershell
   docker compose -f monitoring/compose.yaml up -d
   ```

3. Откройте [Grafana](http://localhost:3000). При первом запуске войдите с логином `admin` и паролем `admin`, затем смените пароль. Готовый дашборд находится в папке **Dictionary service** под названием **Dictionary services**.
4. Откройте [страницу целей Prometheus](http://localhost:9090/targets): у `service1` и `service2` должно быть состояние `UP`. Выполните `go run cmd/cli/main.go --health` из каталога `service1`, чтобы появились первые данные о запросах. Графики скорости и времени ответа заполнятся после нескольких запросов и двух сборов метрик.

Остановить мониторинг: `docker compose -f monitoring/compose.yaml down`. Данные Grafana и Prometheus хранятся в Docker volumes и сохраняются при обычной остановке. Если сервис недоступен, проверьте его терминал и адреса `localhost:8081`/`localhost:8083`; контейнеры обращаются к ним через `host.docker.internal`.

## Команды CLI

Примеры ниже выполняются из каталога `service1` при запущенных обоих сервисах.

### Флаги

| Флаг | Описание | Пример |
|------|----------|--------|
| `--source` | Язык оригинала | `--source zh` |
| `--target` | Целевой язык | `--target ru` |
| `--word` | Слово для перевода | `--word "zha3o"` |
| `--topic` | Тема для получения слов | `--topic animals` |
| `--language` | Один язык | `--language ru` |
| `--languages` | Несколько языков (через запятую) | `--languages ru,en,zh` |
| `--check` | Режим проверки | `--check` |
| `--word` | Исходное слово | `--word "искать"` |
| `--translation` | Перевод пользователя | `--translation "zha3o"` |
| `--language` | Язык оригинала (для проверки) | `--language ru` |
| `--list-languages` | Список языков | `--list-languages` |
| `--list-topics` | Список тем | `--list-topics` |

### Примеры

```bash
# Перевод слова
go run cmd/cli/main.go --source zh --target ru --word "zha3o"

# Проверка перевода
go run cmd/cli/main.go --check --word "утка" --translation "ya1zi" --language ru

# Перевод
go run cmd/cli/main.go --word "утка" --source ru --target zh

# Слова по теме на одном языке (тема обязательно на русском)
go run cmd/cli/main.go --topic животные --languages ru

# Слова по теме на нескольких языках (тема обязательно на русском)
go run cmd/cli/main.go --topic животные --languages ru,en,zh

# Список языков
go run cmd/cli/main.go --list-languages

# Список тем
go run cmd/cli/main.go --list-topics

# Проверка здоровья сервисов
go run cmd/cli/main.go --health
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

### Сервис-1 (`service1/configs/service.json`)
```json
{
  "port": 8081,
  "log_level": "debug",
  "dictionary_service_url": "http://localhost:8083",
  "timeout": 10
}
```

### Сервис-2 (`service2/config/service2.json`)

Пример структуры; подставьте параметры своей БД:

```json
{
  "port": 8083,
  "log_level": "info",
  "timeout": 30,
  "database": {
    "host": "адрес-базы",
    "port": 5432,
    "user": "имя-пользователя",
    "password": "пароль",
    "dbname": "имя-базы",
    "ssl_mode": "disable",
    "timeout": 10
  }
}
```

### CLI (`service1/configs/cli.json`)
```json
{
  "translation_server_url": "http://localhost:8081",
  "timeout": 30,
  "default_format": "table",
  "log_level": "info"
}
```

## Тестирование

Тесты запускаются отдельно в каждом модуле:

```powershell
cd .\service2
go test ./... -v
cd ..\service1
go test ./... -v
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
Ошибка: не удалось подключиться к серверу http://localhost:8081
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

- Go 1.24+
- PostgreSQL 14+
