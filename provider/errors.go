package provider

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrorKind classifies API errors for retry decisions.
type ErrorKind int

const (
	ErrorRetryable     ErrorKind = iota // 429, 5xx
	ErrorAuth                          // 401, 403
	ErrorContextWindow                 // 400 + "context" in message
	ErrorPermanent                     // everything else
)

// APIError represents a classified API error.
type APIError struct {
	StatusCode int
	Kind       ErrorKind
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

var httpStatusPattern = regexp.MustCompile(`HTTP (\d{3})`)

// ClassifyError extracts HTTP status from error strings and classifies them.
// Recognizes patterns like "API error (HTTP 429): rate_limit_error: ..."
func ClassifyError(err error) *APIError {
	if err == nil {
		return nil
	}

	msg := err.Error()
	apiErr := &APIError{Message: msg}

	// Extract HTTP status code from error message.
	if m := httpStatusPattern.FindStringSubmatch(msg); len(m) == 2 {
		apiErr.StatusCode, _ = strconv.Atoi(m[1])
	}

	switch {
	case apiErr.StatusCode == 429 || apiErr.StatusCode >= 500:
		apiErr.Kind = ErrorRetryable
	case apiErr.StatusCode == 401 || apiErr.StatusCode == 403:
		apiErr.Kind = ErrorAuth
	case apiErr.StatusCode == 400 && strings.Contains(strings.ToLower(msg), "context"):
		apiErr.Kind = ErrorContextWindow
	default:
		apiErr.Kind = ErrorPermanent
	}

	return apiErr
}
