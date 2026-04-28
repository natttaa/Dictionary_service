package models

// Error - структура ошибки
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TranslateRequest - запрос на перевод
type TranslateRequest struct {
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
	Word       string `json:"word"`
}

// TranslateResponse - ответ на перевод
type TranslateResponse struct {
	Translation string `json:"translation"`
	Error       *Error `json:"error,omitempty"`
}

// TopicWordsRequest - запрос слов по теме
type TopicWordsRequest struct {
	Topic     string   `json:"topic"`
	Languages []string `json:"languages"`
}

// TopicWordsResponse - ответ со словами по теме
type TopicWordsResponse struct {
	Topic string      `json:"topic"`
	Words []WordEntry `json:"words"`
	Error *Error      `json:"error,omitempty"`
}

// WordEntry - запись слова с переводами на разные языки
type WordEntry struct {
	Translations map[string]string `json:"translations"` // язык -> перевод
}

// CheckTranslationRequest - запрос на проверку перевода
type CheckTranslationRequest struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
	SourceLang  string `json:"source_lang"`
}

// CheckTranslationResponse - ответ на проверку перевода
type CheckTranslationResponse struct {
	CorrectTranslation string `json:"translation,omitempty"`
	Error              *Error `json:"error,omitempty"`
}

// LanguagesResponse - ответ со списком языков
type LanguagesResponse struct {
	Languages []LanguageInfo `json:"languages"`
	Error     *Error         `json:"error,omitempty"`
}

// LanguageInfo - информация о языке
type LanguageInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// TopicsResponse - ответ со списком тем
type TopicsResponse struct {
	Topics []string `json:"topics"`
	Error  *Error   `json:"error,omitempty"`
}

// HealthResponse - ответ проверки здоровья
type HealthResponse struct {
	Status   string `json:"status"`
	Service2 string `json:"service2"`
}
