package errors

import "errors"

// Common errors
var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrValidationFailed   = errors.New("validation failed")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrTimeout            = errors.New("request timeout")
)

// Business errors
var (
	ErrWordNotFound        = errors.New("word not found")
	ErrUnsupportedLang     = errors.New("unsupported language")
	ErrTranslationNotFound = errors.New("translation not found")
)
