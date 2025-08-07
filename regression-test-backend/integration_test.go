//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KamdynS/go-agents/llm"
	"github.com/KamdynS/go-agents/llm/anthropic"
	"github.com/KamdynS/go-agents/llm/openai"
)

const (
	testServerURL = "http://localhost:18080"
	testTimeout   = 60 * time.Second
)

// Test configuration
var (
	skipIntegration = os.Getenv("SKIP_INTEGRATION") == "true"
	testPort        = getEnvDefault("HTTP_PORT", "18080")
)

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestMain(m *testing.M) {
	if skipIntegration {
		fmt.Println("Skipping integration tests (SKIP_INTEGRATION=true)")
		return
	}

	// Verify environment variables are set
	if os.Getenv("OPENAI_API_KEY") == "" || os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println("Skipping integration tests: API keys not configured")
		return
	}

	// Wait for server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverURL := fmt.Sprintf("http://localhost:%s", testPort)
	if !waitForServer(ctx, serverURL) {
		fmt.Println("Test server is not ready")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func waitForServer(ctx context.Context, url string) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			resp, err := client.Get(url + "/health")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return true
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", testPort))
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("Health check returned status %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["status"] != "healthy" {
		t.Fatalf("Server not healthy: %v", health["status"])
	}

	t.Logf("Health check passed: %+v", health)
}

func TestOpenAIDirectClient(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	client, err := openai.NewClient(openai.Config{
		APIKey:    os.Getenv("OPENAI_API_KEY"),
		Model:     llm.ModelGPT4oMini,
		MaxTokens: 100,
		Timeout:   testTimeout,
	})
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	req := &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Say 'Hello from OpenAI!' and nothing else."},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("OpenAI chat failed: %v", err)
	}

	if resp.Content == "" {
		t.Fatal("Empty response from OpenAI")
	}

	if resp.Usage == nil || resp.Usage.TotalTokens == 0 {
		t.Fatal("No usage information returned")
	}

	t.Logf("OpenAI response: %s (tokens: %d, cost: $%.6f)",
		resp.Content, resp.Usage.TotalTokens, resp.Usage.Cost)

	// Verify model and provider
	if resp.Model != llm.ModelGPT4oMini {
		t.Errorf("Expected model %s, got %s", llm.ModelGPT4oMini, resp.Model)
	}

	if resp.Provider != llm.ProviderOpenAI {
		t.Errorf("Expected provider %s, got %s", llm.ProviderOpenAI, resp.Provider)
	}
}

func TestAnthropicDirectClient(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	client, err := anthropic.NewClient(anthropic.Config{
		APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		Model:     llm.ModelClaude35Haiku,
		MaxTokens: 100,
		Timeout:   testTimeout,
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	req := &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "user", Content: "Say 'Hello from Anthropic!' and nothing else."},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Anthropic chat failed: %v", err)
	}

	if resp.Content == "" {
		t.Fatal("Empty response from Anthropic")
	}

	if resp.Usage == nil || resp.Usage.TotalTokens == 0 {
		t.Fatal("No usage information returned")
	}

	t.Logf("Anthropic response: %s (tokens: %d, cost: $%.6f)",
		resp.Content, resp.Usage.TotalTokens, resp.Usage.Cost)
}

func TestStructuredOutputOpenAI(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	client, err := openai.NewClient(openai.Config{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   llm.ModelGPT4oMini,
		Timeout: testTimeout,
	})
	if err != nil {
		t.Fatalf("Failed to create OpenAI client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := openai.StructuredCompletion(
		client,
		ctx,
		"I absolutely love this product! It's fantastic and works perfectly.",
		llm.Sentiment{},
	)
	if err != nil {
		t.Fatalf("Structured completion failed: %v", err)
	}

	if resp.Data.Sentiment != "positive" {
		t.Errorf("Expected positive sentiment, got: %s", resp.Data.Sentiment)
	}

	if resp.Data.Score <= 0 {
		t.Errorf("Expected positive score, got: %f", resp.Data.Score)
	}

	if !resp.Validation.Valid {
		t.Errorf("Validation failed: %v", resp.Validation.Errors)
	}

	t.Logf("OpenAI structured output - Sentiment: %s, Score: %.2f",
		resp.Data.Sentiment, resp.Data.Score)
}

func TestStructuredOutputAnthropic(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	client, err := anthropic.NewClient(anthropic.Config{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		Model:   llm.ModelClaude35Haiku,
		Timeout: testTimeout,
	})
	if err != nil {
		t.Fatalf("Failed to create Anthropic client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	resp, err := anthropic.StructuredCompletion(
		client,
		ctx,
		"This product is terrible and doesn't work at all.",
		llm.Sentiment{},
	)
	if err != nil {
		t.Fatalf("Structured completion failed: %v", err)
	}

	if resp.Data.Sentiment != "negative" {
		t.Errorf("Expected negative sentiment, got: %s", resp.Data.Sentiment)
	}

	if resp.Data.Score >= 0 {
		t.Errorf("Expected negative score, got: %f", resp.Data.Score)
	}

	t.Logf("Anthropic structured output - Sentiment: %s, Score: %.2f",
		resp.Data.Sentiment, resp.Data.Score)
}

func TestServerLLMEndpoint(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	tests := []struct {
		name     string
		provider string
		message  string
	}{
		{"OpenAI", "openai", "Respond with exactly: 'OpenAI test successful'"},
		{"Anthropic", "anthropic", "Respond with exactly: 'Anthropic test successful'"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reqBody := TestRequest{
				Message:  test.message,
				Provider: test.provider,
			}

			jsonBody, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			resp, err := http.Post(
				fmt.Sprintf("http://localhost:%s/test/llm", testPort),
				"application/json",
				bytes.NewBuffer(jsonBody),
			)
			if err != nil {
				t.Fatalf("LLM test request failed: %v", err)
			}
			defer resp.Body.Close()

			var testResp TestResponse
			if err := json.NewDecoder(resp.Body).Decode(&testResp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !testResp.Success {
				t.Fatalf("LLM test failed: %s", testResp.Error)
			}

			if testResp.Response == "" {
				t.Fatal("Empty response")
			}

			if testResp.Usage == nil {
				t.Fatal("No usage information")
			}

			// Check if response contains expected text (case insensitive)
			expectedSubstring := strings.ToLower(test.provider)
			if !strings.Contains(strings.ToLower(testResp.Response), expectedSubstring) {
				t.Logf("Warning: Response doesn't contain '%s': %s", expectedSubstring, testResp.Response)
			}

			t.Logf("%s LLM test - Response: %s (Duration: %s, Tokens: %d)",
				test.name, testResp.Response, testResp.Duration, testResp.Usage.TotalTokens)
		})
	}
}

func TestServerStructuredEndpoint(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	tests := []struct {
		name     string
		provider string
		message  string
		expected string
	}{
		{"OpenAI Positive", "openai", "This is absolutely wonderful and amazing!", "positive"},
		{"Anthropic Negative", "anthropic", "This is terrible and awful!", "negative"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reqBody := TestRequest{
				Message:  test.message,
				Provider: test.provider,
			}

			jsonBody, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			resp, err := http.Post(
				fmt.Sprintf("http://localhost:%s/test/structured", testPort),
				"application/json",
				bytes.NewBuffer(jsonBody),
			)
			if err != nil {
				t.Fatalf("Structured test request failed: %v", err)
			}
			defer resp.Body.Close()

			var testResp TestResponse
			if err := json.NewDecoder(resp.Body).Decode(&testResp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !testResp.Success {
				t.Fatalf("Structured test failed: %s", testResp.Error)
			}

			// Extract structured data from meta
			structuredData, ok := testResp.Meta["structured_data"].(map[string]interface{})
			if !ok {
				t.Fatalf("No structured data found in response: %+v", testResp.Meta)
			}

			sentiment, ok := structuredData["sentiment"].(string)
			if !ok {
				t.Fatalf("No sentiment in structured data: %+v", structuredData)
			}

			if sentiment != test.expected {
				t.Errorf("Expected %s sentiment, got: %s", test.expected, sentiment)
			}

			t.Logf("%s structured test - Sentiment: %s (Duration: %s)",
				test.name, sentiment, testResp.Duration)
		})
	}
}

func TestModelsEndpoint(t *testing.T) {
	if skipIntegration {
		t.Skip("Skipping integration test")
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/test/models", testPort))
	if err != nil {
		t.Fatalf("Models request failed: %v", err)
	}
	defer resp.Body.Close()

	var models map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		t.Fatalf("Failed to decode models response: %v", err)
	}

	// Check that we have model lists
	openaiModels, ok := models["openai_models"].([]interface{})
	if !ok || len(openaiModels) == 0 {
		t.Fatal("No OpenAI models found")
	}

	anthropicModels, ok := models["anthropic_models"].([]interface{})
	if !ok || len(anthropicModels) == 0 {
		t.Fatal("No Anthropic models found")
	}

	modelInfo, ok := models["model_info"].(map[string]interface{})
	if !ok {
		t.Fatal("No model info found")
	}

	t.Logf("Found %d OpenAI models, %d Anthropic models, %d model info entries",
		len(openaiModels), len(anthropicModels), len(modelInfo))

	// Check that model info exists for each model
	for _, modelList := range [][]interface{}{openaiModels, anthropicModels} {
		for _, modelName := range modelList {
			modelStr, ok := modelName.(string)
			if !ok {
				continue
			}
			
			info, exists := modelInfo[modelStr]
			if !exists {
				t.Errorf("No model info for %s", modelStr)
				continue
			}

			infoMap, ok := info.(map[string]interface{})
			if !ok {
				t.Errorf("Invalid model info format for %s", modelStr)
				continue
			}

			// Check required fields
			requiredFields := []string{"provider", "context_size", "input_cost_1k", "output_cost_1k"}
			for _, field := range requiredFields {
				if _, exists := infoMap[field]; !exists {
					t.Errorf("Missing %s in model info for %s", field, modelStr)
				}
			}
		}
	}
}