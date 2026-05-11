package errors

import "errors"

// Базовые ошибки
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrValidationFailed   = errors.New("validation failed")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("request timeout")
)

// Бизнес ошибки
var (
	ErrWordNotFound        = errors.New("word not found")
	ErrUnsupportedLang     = errors.New("unsupported language")
	ErrTranslationNotFound = errors.New("translation not found")
)
