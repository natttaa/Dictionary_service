package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"service1/cmd/cli/client"
	"service1/cmd/cli/config"
	"strings"
)

// printHelp выводит справку по использованию
func printHelp() {
	fmt.Println("Словарно-тренировочный сервис")
	fmt.Println("\nКоманды:")
	fmt.Println("  --health              Проверка здоровья сервиса")
	fmt.Println("  --list-languages      Список языков")
	fmt.Println("  --list-topics         Список тем")
	fmt.Println("  --source LANG         Язык оригинала (zh, ru, en)")
	fmt.Println("  --target LANG         Целевой язык (zh, ru, en)")
	fmt.Println("  --word WORD           Слово для перевода")
	fmt.Println("  --topic TOPIC         Тема для получения слов")
	fmt.Println("  --languages LANGS     Языки через запятую (ru,en,zh)")
	fmt.Println("  --check               Режим проверки перевода")
	fmt.Println("  --original WORD       Исходное слово для проверки")
	fmt.Println("  --translation WORD    Перевод пользователя для проверки")
	fmt.Println("  --lang LANG           Язык оригинала для проверки")
	fmt.Println("  --config PATH         Путь к файлу конфигурации")
	fmt.Println("\nПримеры использования:")
	fmt.Println("  Проверка здоровья:")
	fmt.Println("    go run cmd/cli/main.go --health")
	fmt.Println("\n  Список языков:")
	fmt.Println("    go run cmd/cli/main.go --list-languages")
	fmt.Println("\n  Список тем:")
	fmt.Println("    go run cmd/cli/main.go --list-topics")
	fmt.Println("\n  Перевод слова:")
	fmt.Println("    go run cmd/cli/main.go --source en --target ru --word \"hello\"")
	fmt.Println("\n  Слова по теме (один язык):")
	fmt.Println("    go run cmd/cli/main.go --topic animals --languages ru")
	fmt.Println("\n  Слова по теме (несколько языков):")
	fmt.Println("    go run cmd/cli/main.go --topic food --languages ru,en,zh")
	fmt.Println("\n  Проверка перевода:")
	fmt.Println("    go run cmd/cli/main.go --check --original \"собака\" --translation \"dog\" --lang ru")
}

func main() {
	// Флаги командной строки
	configPath := flag.String("config", "", "Путь к файлу конфигурации")
	source := flag.String("source", "", "Язык оригинала (zh, ru, en)")
	target := flag.String("target", "", "Целевой язык (zh, ru, en)")
	word := flag.String("word", "", "Слово для перевода")
	listLanguages := flag.Bool("list-languages", false, "Список языков")
	listTopics := flag.Bool("list-topics", false, "Список тем")
	health := flag.Bool("health", false, "Проверка здоровья сервиса")

	// Новые флаги
	topic := flag.String("topic", "", "Тема для получения слов")
	languages := flag.String("languages", "", "Языки через запятую (ru,en,zh)")
	check := flag.Bool("check", false, "Режим проверки перевода")
	original := flag.String("original", "", "Исходное слово для проверки")
	translation := flag.String("translation", "", "Перевод пользователя для проверки")
	lang := flag.String("lang", "", "Язык оригинала для проверки")

	flag.Parse()

	// Загрузка конфигурации
	config, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	// Настройка логгера
	logger := config.SetupLogger()

	logger.Info("Запуск CLI",
		slog.String("server_url", config.ServerURL),
		slog.String("log_level", config.LogLevel),
		slog.Duration("timeout", config.Timeout),
	)

	// Создание клиента
	cliClient := client.NewCLIClient(config.ServerURL, config.Timeout, logger)

	// Обработка команд
	switch {
	case *health:
		if err := cliClient.Health(); err != nil {
			logger.Error("Ошибка при проверке здоровья", slog.Any("error", err))
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(1)
		}

	case *listLanguages:
		if err := cliClient.ListLanguages(); err != nil {
			logger.Error("Ошибка при получении списка языков", slog.Any("error", err))
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(1)
		}

	case *listTopics:
		if err := cliClient.ListTopics(); err != nil {
			logger.Error("Ошибка при получении списка тем", slog.Any("error", err))
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(1)
		}

	case *check:
		if *original == "" || *translation == "" || *lang == "" {
			logger.Error("Для проверки перевода укажите --original, --translation и --lang")
			fmt.Fprintln(os.Stderr, "Ошибка: для проверки перевода укажите --original, --translation и --lang")
			os.Exit(1)
		}
		if err := cliClient.CheckTranslation(*original, *translation, *lang); err != nil {
			logger.Error("Ошибка при проверке перевода", slog.Any("error", err))
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(1)
		}

	case *topic != "":
		if *languages == "" {
			logger.Error("Для темы укажите --languages")
			fmt.Fprintln(os.Stderr, "Ошибка: для темы укажите --languages")
			os.Exit(1)
		}
		langList := strings.Split(*languages, ",")
		for i := range langList {
			langList[i] = strings.TrimSpace(langList[i])
		}
		if err := cliClient.GetTopicWords(*topic, langList); err != nil {
			logger.Error("Ошибка при получении слов по теме", slog.Any("error", err))
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(1)
		}

	case *word != "":
		if *source == "" || *target == "" {
			logger.Error("Не указаны языки для перевода")
			fmt.Fprintln(os.Stderr, "Ошибка: для перевода укажите --source и --target")
			os.Exit(1)
		}
		if err := cliClient.Translate(*source, *target, *word); err != nil {
			logger.Error("Ошибка при переводе", slog.Any("error", err))
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			os.Exit(1)
		}

	default:
		printHelp()
	}
}
