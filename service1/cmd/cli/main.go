package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"service1/cmd/cli/client"
	"service1/cmd/cli/config"
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
	fmt.Println("\n  С конфигом:")
	fmt.Println("    go run cmd/cli/main.go --config configs/cli.json --list-languages")
}

func main() {
	// Флаги командной строки
	configPath := flag.String("config", "", "Путь к файлу конфигурации")
	source := flag.String("source", "", "Язык оригинала (zh, ru, en)")
	target := flag.String("target", "", "Целевой язык (zh, ru, en)")
	word := flag.String("word", "", "Слово для перевода")
	listLanguages := flag.Bool("list-languages", false, "Список языков")
	listTopics := flag.Bool("list-topics", false, "Список тем")
	health := flag.Bool("health", false, "Проверка здоровья сервиса") // НОВЫЙ ФЛАГ

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
	case *health: // НОВАЯ КОМАНДА
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
