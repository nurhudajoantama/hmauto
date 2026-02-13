package validation

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	MaxMessageLength = 1000
	MaxTypeLength    = 100
)

var (
	ErrInvalidLevel   = errors.New("invalid alert level")
	ErrMessageTooLong = errors.New("message exceeds maximum length")
	ErrTypeTooLong    = errors.New("type exceeds maximum length")
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrEmptyType      = errors.New("type cannot be empty")
)

// ValidLevel checks if the level is one of the allowed values
func ValidLevel(level string, allowedLevels []string) bool {
	for _, allowed := range allowedLevels {
		if level == allowed {
			return true
		}
	}
	return false
}

// SanitizeString removes potentially dangerous characters and trims whitespace
func SanitizeString(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	return input
}

// ValidateAlertMessage validates an alert message
func ValidateAlertMessage(message string) error {
	message = SanitizeString(message)
	
	if message == "" {
		return ErrEmptyMessage
	}
	
	if utf8.RuneCountInString(message) > MaxMessageLength {
		return ErrMessageTooLong
	}
	
	return nil
}

// ValidateAlertType validates an alert type
func ValidateAlertType(alertType string) error {
	alertType = SanitizeString(alertType)
	
	if alertType == "" {
		return ErrEmptyType
	}
	
	if utf8.RuneCountInString(alertType) > MaxTypeLength {
		return ErrTypeTooLong
	}
	
	return nil
}

// ValidateAlertLevel validates an alert level
func ValidateAlertLevel(level string, allowedLevels []string) error {
	level = SanitizeString(level)
	
	if !ValidLevel(level, allowedLevels) {
		return ErrInvalidLevel
	}
	
	return nil
}
