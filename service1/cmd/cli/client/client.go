package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"service1/models"
	"strings"
	"text/tabwriter"
	"time"
)

// Language names mapping
var languageNames = map[string]string{
	"zh": "Китайский",
	"ru": "Русский",
	"en": "Английский",
}

// getLanguageName возвращает название языка по коду
func getLanguageName(code string) string {
	if name, ok := languageNames[code]; ok {
		return name
	}
	return code
}

// CLIClient представляет HTTP клиент для взаимодействия с сервисом
type CLIClient struct {
	serverURL string
	client    *http.Client
	logger    *slog.Logger
}

// NewCLIClient создает новый экземпляр CLI клиента
func NewCLIClient(serverURL string, timeout time.Duration, logger *slog.Logger) *CLIClient {
	return &CLIClient{
		serverURL: serverURL,
		client: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// doRequest выполняет HTTP запрос
func (c *CLIClient) doRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ошибка маршалинга: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, c.serverURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Выполнение запроса",
		slog.String("method", method),
		slog.String("url", c.serverURL+endpoint),
	)

	return c.client.Do(req)
}

// Health проверяет состояние сервиса
func (c *CLIClient) Health() error {
	c.logger.Debug("Запрос проверки здоровья сервиса")

	resp, err := c.doRequest("GET", "/api/v1/health", nil)
	if err != nil {
		return fmt.Errorf("ошибка подключения к серверу: %w", err)
	}
	defer resp.Body.Close()

	var healthResp models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Вывод статуса здоровья
	fmt.Println("Состояние сервиса:")
	fmt.Println(strings.Repeat("-", 30))

	fmt.Printf("Статус сервиса: %s\n", healthResp.Status)

	// Отображаем статус словарного сервиса

	fmt.Printf("Словарный сервис: %s\n", healthResp.Service2)

	// Дополнительная информация о состоянии
	if healthResp.Status == "healthy" {
		fmt.Println("\nСервис работает в штатном режиме")
	} else {
		fmt.Println("\nСервис недоступен, проверьте подключение")
	}

	c.logger.Debug("Проверка здоровья выполнена",
		slog.String("service_status", healthResp.Status),
		slog.String("dictionary_status", healthResp.Service2),
	)

	return nil
}

// ListLanguages выводит список доступных языков
func (c *CLIClient) ListLanguages() error {
	c.logger.Debug("Запрос списка языков")

	resp, err := c.doRequest("GET", "/api/v1/languages", nil)
	if err != nil {
		return fmt.Errorf("ошибка подключения к серверу: %w", err)
	}
	defer resp.Body.Close()

	var langsResp models.LanguagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&langsResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if langsResp.Error != nil {
		return fmt.Errorf("%s: %s", langsResp.Error.Code, langsResp.Error.Message)
	}

	// Вывод языков
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Код\tЯзык")
	fmt.Fprintln(w, strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 15))

	for _, lang := range langsResp.Languages {
		fmt.Fprintf(w, "%s\t%s\n", lang.Code, lang.Name)
	}
	w.Flush()

	c.logger.Debug("Список языков успешно получен", slog.Int("count", len(langsResp.Languages)))

	return nil
}

// ListTopics выводит список доступных тем
func (c *CLIClient) ListTopics() error {
	c.logger.Debug("Запрос списка тем")

	resp, err := c.doRequest("GET", "/api/v1/topics", nil)
	if err != nil {
		return fmt.Errorf("ошибка подключения к серверу: %w", err)
	}
	defer resp.Body.Close()

	var topicsResp models.TopicsResponse
	if err := json.NewDecoder(resp.Body).Decode(&topicsResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if topicsResp.Error != nil {
		return fmt.Errorf("%s: %s", topicsResp.Error.Code, topicsResp.Error.Message)
	}

	// Вывод тем
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Темы")
	fmt.Fprintln(w, strings.Repeat("-", 20))

	for _, topic := range topicsResp.Topics {
		fmt.Fprintf(w, "%s\n", topic)
	}
	w.Flush()

	c.logger.Debug("Список тем успешно получен", slog.Int("count", len(topicsResp.Topics)))

	return nil
}

// Translate выполняет перевод слова
func (c *CLIClient) Translate(source, target, word string) error {
	c.logger.Debug("Запрос на перевод",
		slog.String("source", source),
		slog.String("target", target),
		slog.String("word", word),
	)

	req := models.TranslateRequest{
		SourceLang: source,
		TargetLang: target,
		Word:       word,
	}

	resp, err := c.doRequest("POST", "/api/v1/translate", req)
	if err != nil {
		return fmt.Errorf("ошибка подключения к серверу: %w", err)
	}
	defer resp.Body.Close()

	var translateResp models.TranslateResponse
	if err := json.NewDecoder(resp.Body).Decode(&translateResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if translateResp.Error != nil {
		return fmt.Errorf("%s: %s", translateResp.Error.Code, translateResp.Error.Message)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\n", getLanguageName(source), getLanguageName(target))
	fmt.Fprintf(w, "%s\t%s\n", strings.Repeat("-", 15), strings.Repeat("-", 15))
	fmt.Fprintf(w, "%s\t%s\n", word, translateResp.Translation)
	w.Flush()

	c.logger.Debug("Перевод выполнен успешно",
		slog.String("source", source),
		slog.String("target", target),
		slog.String("word", word),
		slog.String("translation", translateResp.Translation),
	)

	return nil
}

// GetTopicWords получает слова по теме для одного или нескольких языков
func (c *CLIClient) GetTopicWords(topic string, languages []string) error {
	c.logger.Debug("Запрос слов по теме",
		slog.String("topic", topic),
		slog.Any("languages", languages),
	)

	req := models.TopicWordsRequest{
		Topic:     topic,
		Languages: languages,
	}

	resp, err := c.doRequest("POST", "/api/v1/topics/words", req)
	if err != nil {
		return fmt.Errorf("ошибка подключения к серверу: %w", err)
	}
	defer resp.Body.Close()

	var topicResp models.TopicWordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&topicResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if topicResp.Error != nil {
		return fmt.Errorf("%s: %s", topicResp.Error.Code, topicResp.Error.Message)
	}

	// Если один язык - выводим списком
	if len(languages) == 1 {
		fmt.Printf("Тема: %s\n", topicResp.Topic)
		fmt.Printf("Язык: %s\n", getLanguageName(languages[0]))
		fmt.Println(strings.Repeat("-", 30))

		for _, entry := range topicResp.Words {
			for _, translation := range entry.Translations {
				fmt.Printf("%s\n", translation)
			}
		}
	} else {
		// Если несколько языков - выводим таблицей
		fmt.Printf("Тема: %s\n", topicResp.Topic)
		fmt.Printf("Языки: ")
		for i, lang := range languages {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(getLanguageName(lang))
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", 50))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

		// Заголовки
		headers := make([]string, len(languages))
		for i, lang := range languages {
			headers[i] = getLanguageName(lang)
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))

		// Разделитель
		separator := make([]string, len(languages))
		for i := range separator {
			separator[i] = strings.Repeat("-", 15)
		}
		fmt.Fprintln(w, strings.Join(separator, "\t"))

		// Данные
		for _, entry := range topicResp.Words {
			row := make([]string, len(languages))
			for i, lang := range languages {
				if translation, ok := entry.Translations[lang]; ok {
					row[i] = translation
				} else {
					row[i] = "-"
				}
			}
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
		w.Flush()
	}

	c.logger.Debug("Слова по теме получены",
		slog.String("topic", topicResp.Topic),
		slog.Int("count", len(topicResp.Words)),
	)

	return nil
}

// CheckTranslation проверяет перевод пользователя
func (c *CLIClient) CheckTranslation(word, translation, sourceLang string) error {
	c.logger.Debug("Запрос на проверку перевода",
		slog.String("word", word),
		slog.String("translation", translation),
		slog.String("source_lang", sourceLang),
	)

	req := models.CheckTranslationRequest{
		Word:        word,
		Translation: translation,
		SourceLang:  sourceLang,
	}

	resp, err := c.doRequest("POST", "/api/v1/check-translation", req)
	if err != nil {
		return fmt.Errorf("ошибка подключения к серверу: %w", err)
	}
	defer resp.Body.Close()

	var checkResp models.CheckTranslationResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if checkResp.Error != nil {
		return fmt.Errorf("%s: %s", checkResp.Error.Code, checkResp.Error.Message)
	}

	fmt.Printf("Проверка перевода:\n")
	fmt.Printf("  Исходное слово: %s (%s)\n", word, getLanguageName(sourceLang))
	fmt.Printf("  Ваш перевод: %s\n", translation)
	fmt.Printf("  Правильный перевод: %s\n", checkResp.CorrectTranslation)

	if strings.EqualFold(strings.TrimSpace(translation), strings.TrimSpace(checkResp.CorrectTranslation)) {
		fmt.Println("\nПравильно!")
	} else {
		fmt.Println("\nНеправильно!")
	}

	c.logger.Debug("Проверка перевода выполнена",
		slog.String("word", word),
		slog.String("user_translation", translation),
		slog.String("correct_translation", checkResp.CorrectTranslation),
	)

	return nil
}
