package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewRetrier(t *testing.T) {
	config := RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		BackoffFactor: 2.0,
	}

	retrier := NewRetrier(config)

	if retrier.config.MaxRetries != config.MaxRetries {
		t.Errorf("Expected MaxRetries %d, got %d", config.MaxRetries, retrier.config.MaxRetries)
	}

	if retrier.config.InitialDelay != config.InitialDelay {
		t.Errorf("Expected InitialDelay %v, got %v", config.InitialDelay, retrier.config.InitialDelay)
	}

	if retrier.rand == nil {
		t.Error("Expected rand to be initialized")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries <= 0 {
		t.Errorf("Expected positive MaxRetries, got %d", config.MaxRetries)
	}

	if config.InitialDelay <= 0 {
		t.Errorf("Expected positive InitialDelay, got %v", config.InitialDelay)
	}

	if config.MaxDelay <= config.InitialDelay {
		t.Errorf("Expected MaxDelay (%v) > InitialDelay (%v)", config.MaxDelay, config.InitialDelay)
	}

	if config.BackoffFactor <= 1.0 {
		t.Errorf("Expected BackoffFactor > 1.0, got %f", config.BackoffFactor)
	}
}

func TestExecute_Success(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
	}
	retrier := NewRetrier(config)

	ctx := context.Background()
	expectedResult := "success"

	operation := func(ctx context.Context, attempt int) (string, error) {
		return expectedResult, nil
	}

	result, err := Execute(retrier, ctx, operation)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("Expected result %s, got %s", expectedResult, result)
	}
}

func TestExecute_EventualSuccess(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
	}
	retrier := NewRetrier(config)

	ctx := context.Background()
	attempts := 0
	expectedResult := "success"

	operation := func(ctx context.Context, attempt int) (string, error) {
		attempts++
		if attempts < 3 {
			return "", NewLLMError(ProviderOpenAI, ErrorTypeRateLimit, "rate limited")
		}
		return expectedResult, nil
	}

	result, err := Execute(retrier, ctx, operation)
	if err != nil {
		t.Fatalf("Expected eventual success, got error: %v", err)
	}

	if result != expectedResult {
		t.Errorf("Expected result %s, got %s", expectedResult, result)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestExecute_NonRetryableError(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
	}
	retrier := NewRetrier(config)

	ctx := context.Background()
	attempts := 0
	expectedError := NewLLMError(ProviderOpenAI, ErrorTypeAuthentication, "invalid API key")

	operation := func(ctx context.Context, attempt int) (string, error) {
		attempts++
		return "", expectedError
	}

	_, err := Execute(retrier, ctx, operation)
	if err == nil {
		t.Fatal("Expected error, got success")
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestExecute_MaxRetriesExceeded(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
	}
	retrier := NewRetrier(config)

	ctx := context.Background()
	attempts := 0
	retryableError := NewLLMError(ProviderOpenAI, ErrorTypeServerError, "server error")

	operation := func(ctx context.Context, attempt int) (string, error) {
		attempts++
		return "", retryableError
	}

	_, err := Execute(retrier, ctx, operation)
	if err == nil {
		t.Fatal("Expected error after max retries, got success")
	}

	if attempts != 3 { // Initial attempt + 2 retries
		t.Errorf("Expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}

	if !strings.Contains(err.Error(), "operation failed after") {
		t.Errorf("Expected retry exhaustion error, got: %v", err)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
	}
	retrier := NewRetrier(config)

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	operation := func(ctx context.Context, attempt int) (string, error) {
		attempts++
		if attempts == 1 {
			cancel() // Cancel context after first attempt
		}
		return "", NewLLMError(ProviderOpenAI, ErrorTypeRateLimit, "rate limited")
	}

	_, err := Execute(retrier, ctx, operation)
	if err == nil {
		t.Fatal("Expected context cancellation error, got success")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
}

func TestExecuteSimple(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
	}
	retrier := NewRetrier(config)

	ctx := context.Background()
	attempts := 0

	operation := func(ctx context.Context, attempt int) error {
		attempts++
		if attempts < 2 {
			return NewLLMError(ProviderOpenAI, ErrorTypeRateLimit, "rate limited")
		}
		return nil
	}

	err := retrier.ExecuteSimple(ctx, operation)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestShouldRetry(t *testing.T) {
	config := RetryConfig{
		MaxRetries: 3,
		RetryableErrors: []string{"timeout", "connection"},
	}
	retrier := NewRetrier(config)

	tests := []struct {
		name     string
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "LLM retryable error",
			err:      NewLLMError(ProviderOpenAI, ErrorTypeRateLimit, "rate limited"),
			attempt:  1,
			expected: true,
		},
		{
			name:     "LLM non-retryable error",
			err:      NewLLMError(ProviderOpenAI, ErrorTypeAuthentication, "invalid key"),
			attempt:  1,
			expected: false,
		},
		{
			name:     "Max attempts reached",
			err:      NewLLMError(ProviderOpenAI, ErrorTypeRateLimit, "rate limited"),
			attempt:  3,
			expected: false,
		},
		{
			name:     "Configured retryable error",
			err:      errors.New("connection timeout occurred"),
			attempt:  1,
			expected: true,
		},
		{
			name:     "Non-configured error",
			err:      errors.New("some other error"),
			attempt:  1,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := retrier.shouldRetry(test.err, test.attempt)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	config := RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      time.Second,
		BackoffFactor: 2.0,
	}
	retrier := NewRetrier(config)

	tests := []struct {
		name    string
		attempt int
		err     error
		minTime time.Duration
		maxTime time.Duration
	}{
		{
			name:    "First retry",
			attempt: 0,
			err:     errors.New("error"),
			minTime: 75 * time.Millisecond,  // 100ms - 25% jitter
			maxTime: 125 * time.Millisecond, // 100ms + 25% jitter
		},
		{
			name:    "Second retry",
			attempt: 1,
			err:     errors.New("error"),
			minTime: 150 * time.Millisecond, // 200ms - 25% jitter
			maxTime: 250 * time.Millisecond, // 200ms + 25% jitter
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delay := retrier.calculateDelay(test.attempt, test.err)
			
			if delay < test.minTime || delay > test.maxTime {
				t.Errorf("Expected delay between %v and %v, got %v", 
					test.minTime, test.maxTime, delay)
			}
		})
	}
}

func TestCalculateDelay_RetryAfter(t *testing.T) {
	config := RetryConfig{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      time.Second,
		BackoffFactor: 2.0,
	}
	retrier := NewRetrier(config)

	// Test with LLM error that specifies retry after
	llmErr := &LLMError{
		Type:       ErrorTypeRateLimit,
		RetryAfter: 5, // 5 seconds
	}

	delay := retrier.calculateDelay(1, llmErr)
	expected := 5 * time.Second

	if delay != expected {
		t.Errorf("Expected delay %v, got %v", expected, delay)
	}
}

func TestCalculateDelay_MaxDelay(t *testing.T) {
	config := RetryConfig{
		InitialDelay:  time.Second,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 10.0, // Large factor to exceed max delay
	}
	retrier := NewRetrier(config)

	delay := retrier.calculateDelay(3, errors.New("error"))
	
	// Should be capped at MaxDelay
	if delay > config.MaxDelay {
		t.Errorf("Expected delay <= %v, got %v", config.MaxDelay, delay)
	}
}

func TestNewStatTrackingRetrier(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
	}

	statRetrier := NewStatTrackingRetrier(config)

	if statRetrier.Retrier == nil {
		t.Error("Expected Retrier to be initialized")
	}

	if statRetrier.stats.ErrorTypes == nil {
		t.Error("Expected ErrorTypes map to be initialized")
	}

	if statRetrier.stats.TotalAttempts != 0 {
		t.Error("Expected initial TotalAttempts to be 0")
	}
}

func TestExecuteWithStats_Success(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
	}
	statRetrier := NewStatTrackingRetrier(config)

	ctx := context.Background()
	operation := func(ctx context.Context, attempt int) (string, error) {
		return "success", nil
	}

	result, err := ExecuteWithStats(statRetrier, ctx, operation)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got %s", result)
	}

	stats := statRetrier.GetStats()
	if stats.TotalAttempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", stats.TotalAttempts)
	}

	if stats.Successful != 1 {
		t.Errorf("Expected 1 successful, got %d", stats.Successful)
	}

	if stats.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", stats.Failed)
	}
}

func TestExecuteWithStats_EventualSuccess(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
	}
	statRetrier := NewStatTrackingRetrier(config)

	ctx := context.Background()
	attempts := 0

	operation := func(ctx context.Context, attempt int) (string, error) {
		attempts++
		if attempts < 3 {
			return "", NewLLMError(ProviderOpenAI, ErrorTypeRateLimit, "rate limited")
		}
		return "success", nil
	}

	result, err := ExecuteWithStats(statRetrier, ctx, operation)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result != "success" {
		t.Errorf("Expected 'success', got %s", result)
	}

	stats := statRetrier.GetStats()
	if stats.TotalAttempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", stats.TotalAttempts)
	}

	if stats.Successful != 1 {
		t.Errorf("Expected 1 successful, got %d", stats.Successful)
	}

	if stats.Failed != 0 {
		t.Errorf("Expected 0 failed, got %d", stats.Failed)
	}

	// Check error type tracking
	if stats.ErrorTypes["rate_limit_exceeded"] != 2 {
		t.Errorf("Expected 2 rate limit errors, got %d", stats.ErrorTypes["rate_limit_exceeded"])
	}
}

func TestExecuteWithStats_Failure(t *testing.T) {
	config := RetryConfig{
		MaxRetries:   2,
		InitialDelay: time.Millisecond,
	}
	statRetrier := NewStatTrackingRetrier(config)

	ctx := context.Background()
	operation := func(ctx context.Context, attempt int) (string, error) {
		return "", NewLLMError(ProviderOpenAI, ErrorTypeAuthentication, "auth failed")
	}

	_, err := ExecuteWithStats(statRetrier, ctx, operation)
	if err == nil {
		t.Fatal("Expected error, got success")
	}

	stats := statRetrier.GetStats()
	if stats.TotalAttempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", stats.TotalAttempts)
	}

	if stats.Successful != 0 {
		t.Errorf("Expected 0 successful, got %d", stats.Successful)
	}

	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed, got %d", stats.Failed)
	}

	if stats.LastError == "" {
		t.Error("Expected LastError to be set")
	}
}

func TestResetStats(t *testing.T) {
	config := RetryConfig{MaxRetries: 1}
	statRetrier := NewStatTrackingRetrier(config)

	// Execute an operation to generate stats
	ctx := context.Background()
	operation := func(ctx context.Context, attempt int) (string, error) {
		return "", errors.New("test error")
	}
	
	ExecuteWithStats(statRetrier, ctx, operation)

	// Verify stats were recorded
	stats := statRetrier.GetStats()
	if stats.TotalAttempts == 0 {
		t.Fatal("Expected stats to be recorded before reset")
	}

	// Reset stats
	statRetrier.ResetStats()

	// Verify stats were reset
	stats = statRetrier.GetStats()
	if stats.TotalAttempts != 0 {
		t.Errorf("Expected TotalAttempts to be 0 after reset, got %d", stats.TotalAttempts)
	}

	if stats.Successful != 0 {
		t.Errorf("Expected Successful to be 0 after reset, got %d", stats.Successful)
	}

	if stats.Failed != 0 {
		t.Errorf("Expected Failed to be 0 after reset, got %d", stats.Failed)
	}

	if stats.LastError != "" {
		t.Errorf("Expected LastError to be empty after reset, got %s", stats.LastError)
	}

	if len(stats.ErrorTypes) != 0 {
		t.Errorf("Expected ErrorTypes to be empty after reset, got %v", stats.ErrorTypes)
	}
}