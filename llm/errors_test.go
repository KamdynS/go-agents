package llm

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestLLMError(t *testing.T) {
	tests := []struct {
		name         string
		provider     Provider
		errorType    ErrorType
		message      string
		code         string
		expectedText string
	}{
		{
			name:         "Basic error",
			provider:     ProviderOpenAI,
			errorType:    ErrorTypeRateLimit,
			message:      "Rate limit exceeded",
			expectedText: "openai: Rate limit exceeded",
		},
		{
			name:         "Error with code",
			provider:     ProviderAnthropic,
			errorType:    ErrorTypeInvalidRequest,
			message:      "Invalid request",
			code:         "invalid_request_error",
			expectedText: "anthropic [invalid_request_error]: Invalid request",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := &LLMError{
				Type:     test.errorType,
				Message:  test.message,
				Code:     test.code,
				Provider: test.provider,
			}

			if err.Error() != test.expectedText {
				t.Errorf("Expected error text %q, got %q", test.expectedText, err.Error())
			}
		})
	}
}

func TestNewLLMError(t *testing.T) {
	provider := ProviderOpenAI
	errorType := ErrorTypeRateLimit
	message := "Rate limit exceeded"

	err := NewLLMError(provider, errorType, message)

	if err.Provider != provider {
		t.Errorf("Expected provider %s, got %s", provider, err.Provider)
	}

	if err.Type != errorType {
		t.Errorf("Expected error type %s, got %s", errorType, err.Type)
	}

	if err.Message != message {
		t.Errorf("Expected message %q, got %q", message, err.Message)
	}

	// Check that retryable is set correctly
	expectedRetryable := isRetryableError(errorType)
	if err.Retryable != expectedRetryable {
		t.Errorf("Expected retryable %v, got %v", expectedRetryable, err.Retryable)
	}
}

func TestNewLLMErrorWithCause(t *testing.T) {
	provider := ProviderAnthropic
	errorType := ErrorTypeConnectionError
	message := "Connection failed"
	cause := fmt.Errorf("network timeout")

	err := NewLLMErrorWithCause(provider, errorType, message, cause)

	if err.Cause != cause {
		t.Errorf("Expected cause %v, got %v", cause, err.Cause)
	}

	if err.Unwrap() != cause {
		t.Errorf("Expected unwrap to return %v, got %v", cause, err.Unwrap())
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		retryable bool
	}{
		{ErrorTypeRateLimit, true},
		{ErrorTypeServerError, true},
		{ErrorTypeTimeout, true},
		{ErrorTypeConnectionError, true},
		{ErrorTypeInvalidRequest, false},
		{ErrorTypeAuthentication, false},
		{ErrorTypePermission, false},
		{ErrorTypeNotFound, false},
		{ErrorTypeInsufficientQuota, false},
		{ErrorTypeInvalidModel, false},
		{ErrorTypeContextLength, false},
		{ErrorTypeContentFilter, false},
		{ErrorTypeValidationError, false},
		{ErrorTypeJSONParsingError, false},
		{ErrorTypeUnknown, false},
	}

	for _, test := range tests {
		t.Run(string(test.errorType), func(t *testing.T) {
			retryable := isRetryableError(test.errorType)
			if retryable != test.retryable {
				t.Errorf("Expected %s to be retryable=%v, got %v", 
					test.errorType, test.retryable, retryable)
			}
		})
	}
}

func TestParseHTTPError(t *testing.T) {
	tests := []struct {
		name         string
		provider     Provider
		statusCode   int
		body         string
		expectedType ErrorType
		retryable    bool
	}{
		{
			name:         "Bad Request",
			provider:     ProviderOpenAI,
			statusCode:   http.StatusBadRequest,
			body:         "",
			expectedType: ErrorTypeInvalidRequest,
			retryable:    false,
		},
		{
			name:         "Unauthorized",
			provider:     ProviderAnthropic,
			statusCode:   http.StatusUnauthorized,
			body:         "",
			expectedType: ErrorTypeAuthentication,
			retryable:    false,
		},
		{
			name:         "Rate Limited",
			provider:     ProviderOpenAI,
			statusCode:   http.StatusTooManyRequests,
			body:         "",
			expectedType: ErrorTypeRateLimit,
			retryable:    true,
		},
		{
			name:         "Server Error",
			provider:     ProviderAnthropic,
			statusCode:   http.StatusInternalServerError,
			body:         "",
			expectedType: ErrorTypeServerError,
			retryable:    true,
		},
		{
			name:         "Unknown Status",
			provider:     ProviderOpenAI,
			statusCode:   418, // I'm a teapot
			body:         "",
			expectedType: ErrorTypeUnknown,
			retryable:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ParseHTTPError(test.provider, test.statusCode, test.body)

			if err.Type != test.expectedType {
				t.Errorf("Expected error type %s, got %s", test.expectedType, err.Type)
			}

			if err.Provider != test.provider {
				t.Errorf("Expected provider %s, got %s", test.provider, err.Provider)
			}

			if err.HTTPStatus != test.statusCode {
				t.Errorf("Expected status %d, got %d", test.statusCode, err.HTTPStatus)
			}

			if err.Retryable != test.retryable {
				t.Errorf("Expected retryable %v, got %v", test.retryable, err.Retryable)
			}
		})
	}
}

func TestExtractSpecificError(t *testing.T) {
	tests := []struct {
		name         string
		provider     Provider
		body         string
		expectedType *ErrorType
	}{
		{
			name:         "Rate limit in body",
			provider:     ProviderOpenAI,
			body:         "Error: rate limit exceeded",
			expectedType: &[]ErrorType{ErrorTypeRateLimit}[0],
		},
		{
			name:         "Insufficient quota",
			provider:     ProviderAnthropic,
			body:         "insufficient quota remaining",
			expectedType: &[]ErrorType{ErrorTypeInsufficientQuota}[0],
		},
		{
			name:         "Context length exceeded",
			provider:     ProviderOpenAI,
			body:         "Request exceeds context length limit",
			expectedType: &[]ErrorType{ErrorTypeContextLength}[0],
		},
		{
			name:         "Content filter",
			provider:     ProviderAnthropic,
			body:         "Content filtered by safety system",
			expectedType: &[]ErrorType{ErrorTypeContentFilter}[0],
		},
		{
			name:         "Invalid model",
			provider:     ProviderOpenAI,
			body:         "The model 'invalid-model' does not exist",
			expectedType: &[]ErrorType{ErrorTypeInvalidModel}[0],
		},
		{
			name:         "No specific error",
			provider:     ProviderOpenAI,
			body:         "Some random error message",
			expectedType: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := extractSpecificError(test.provider, test.body)

			if test.expectedType == nil {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Expected error of type %s, got nil", *test.expectedType)
			}

			if err.Type != *test.expectedType {
				t.Errorf("Expected error type %s, got %s", *test.expectedType, err.Type)
			}

			if err.Provider != test.provider {
				t.Errorf("Expected provider %s, got %s", test.provider, err.Provider)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		maxLength int
		expected  string
	}{
		{
			name:      "Short body",
			body:      "Hello",
			maxLength: 10,
			expected:  "Hello",
		},
		{
			name:      "Exact length",
			body:      "Hello World",
			maxLength: 11,
			expected:  "Hello World",
		},
		{
			name:      "Long body",
			body:      "This is a very long error message that should be truncated",
			maxLength: 20,
			expected:  "This is a very long ...",
		},
		{
			name:      "Zero length",
			body:      "Hello",
			maxLength: 0,
			expected:  "...",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := truncateBody(test.body, test.maxLength)
			if result != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, result)
			}
		})
	}
}

func TestIsLLMError(t *testing.T) {
	// Test with LLMError
	llmErr := &LLMError{
		Type:    ErrorTypeRateLimit,
		Message: "Rate limit",
	}

	err, isLLM := IsLLMError(llmErr)
	if !isLLM {
		t.Error("Expected IsLLMError to return true for LLMError")
	}
	if err != llmErr {
		t.Error("Expected returned error to be the same instance")
	}

	// Test with regular error
	regularErr := fmt.Errorf("regular error")
	err, isLLM = IsLLMError(regularErr)
	if isLLM {
		t.Error("Expected IsLLMError to return false for regular error")
	}
	if err != nil {
		t.Error("Expected returned error to be nil for regular error")
	}
}

func TestErrorCheckers(t *testing.T) {
	tests := []struct {
		name        string
		errorType   ErrorType
		checker     func(error) bool
		shouldMatch bool
	}{
		{"Rate limit error", ErrorTypeRateLimit, IsRateLimitError, true},
		{"Not rate limit", ErrorTypeAuthentication, IsRateLimitError, false},
		{"Context length error", ErrorTypeContextLength, IsContextLengthError, true},
		{"Not context length", ErrorTypeRateLimit, IsContextLengthError, false},
		{"Auth error", ErrorTypeAuthentication, IsAuthenticationError, true},
		{"Not auth error", ErrorTypeRateLimit, IsAuthenticationError, false},
		{"Retryable error", ErrorTypeServerError, IsRetryableError, true},
		{"Not retryable", ErrorTypeAuthentication, IsRetryableError, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test with LLMError
			llmErr := &LLMError{Type: test.errorType}
			result := test.checker(llmErr)
			if result != test.shouldMatch {
				t.Errorf("Expected %v, got %v for LLMError", test.shouldMatch, result)
			}

			// Test with regular error
			regularErr := fmt.Errorf("regular error")
			result = test.checker(regularErr)
			if result {
				t.Error("Expected false for regular error")
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		value    interface{}
		message  string
		expected string
	}{
		{
			name:     "Field error",
			field:    "temperature",
			value:    2.5,
			message:  "must be between 0 and 1",
			expected: "validation error on field 'temperature': must be between 0 and 1",
		},
		{
			name:     "General error",
			field:    "",
			value:    nil,
			message:  "invalid format",
			expected: "validation error: invalid format",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := &ValidationError{
				Field:   test.field,
				Value:   test.value,
				Message: test.message,
			}

			if err.Error() != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, err.Error())
			}
		})
	}
}

func TestMultiValidationError(t *testing.T) {
	multi := &MultiValidationError{}

	// Test empty
	if multi.HasErrors() {
		t.Error("Expected no errors initially")
	}

	if multi.ErrorOrNil() != nil {
		t.Error("Expected ErrorOrNil to return nil when no errors")
	}

	// Add errors
	multi.Add("field1", "value1", "error1")
	multi.Add("field2", "value2", "error2")

	if !multi.HasErrors() {
		t.Error("Expected to have errors after adding")
	}

	if multi.ErrorOrNil() == nil {
		t.Error("Expected ErrorOrNil to return error when errors exist")
	}

	if len(multi.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(multi.Errors))
	}

	// Test single error message
	single := &MultiValidationError{}
	single.Add("test", "value", "message")
	if !strings.Contains(single.Error(), "message") {
		t.Error("Single error should contain the message")
	}

	// Test multiple errors message
	errorText := multi.Error()
	if !strings.Contains(errorText, "2 validation errors") {
		t.Errorf("Expected multi-error message, got: %s", errorText)
	}
}