package llm

import (
	"testing"
)

func TestModelConstants(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"OpenAI GPT-4o", ModelGPT4o},
		{"OpenAI GPT-4o Mini", ModelGPT4oMini},
		{"OpenAI GPT-4 Turbo", ModelGPT4Turbo},
		{"OpenAI GPT-3.5 Turbo", ModelGPT35Turbo},
		{"Anthropic Claude 3.5 Sonnet", ModelClaude35Sonnet},
		{"Anthropic Claude 3.5 Haiku", ModelClaude35Haiku},
		{"Anthropic Claude 3 Opus", ModelClaudeOpus},
		{"Anthropic Claude 3 Sonnet", ModelClaudeSonnet},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.model == "" {
				t.Errorf("Model constant %s is empty", test.name)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		expectedExists   bool
		expectedProvider Provider
	}{
		{"OpenAI GPT-4o", ModelGPT4o, true, ProviderOpenAI},
		{"OpenAI GPT-4o Mini", ModelGPT4oMini, true, ProviderOpenAI},
		{"Anthropic Claude 3.5 Sonnet", ModelClaude35Sonnet, true, ProviderAnthropic},
		{"Anthropic Claude 3.5 Haiku", ModelClaude35Haiku, true, ProviderAnthropic},
		{"Invalid Model", "invalid-model", false, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model, err := GetModel(test.model)

			exists := (err == nil)
			if exists != test.expectedExists {
				t.Errorf("Expected exists=%v, got %v", test.expectedExists, exists)
			}

			if !exists {
				return // Skip further checks for non-existent models
			}

			if model.Name != test.model {
				t.Errorf("Expected model name %s, got %s", test.model, model.Name)
			}

			if model.Provider != test.expectedProvider {
				t.Errorf("Expected provider %s, got %s", test.expectedProvider, model.Provider)
			}

			// Check required fields
			if model.ContextSize <= 0 {
				t.Errorf("Model %s has invalid context size: %d", test.model, model.ContextSize)
			}

			if model.InputCost < 0 {
				t.Errorf("Model %s has negative input cost: %f", test.model, model.InputCost)
			}

			if model.OutputCost < 0 {
				t.Errorf("Model %s has negative output cost: %f", test.model, model.OutputCost)
			}
		})
	}
}

func TestValidateModel(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		shouldError bool
	}{
		{"Valid OpenAI Model", ModelGPT4o, false},
		{"Valid Anthropic Model", ModelClaude35Sonnet, false},
		{"Invalid Model", "invalid-model", true},
		{"Empty Model", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateModel(test.model)

			if test.shouldError && err == nil {
				t.Error("Expected error but got none")
			}

			if !test.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestModelEstimateCost(t *testing.T) {
	model, err := GetModel(ModelGPT4oMini)
	if err != nil {
		t.Fatal("GPT-4o Mini model not found")
	}

	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		expectCost   bool
	}{
		{"Zero tokens", 0, 0, true},
		{"Input only", 1000, 0, true},
		{"Output only", 0, 500, true},
		{"Both tokens", 1000, 500, true},
		{"Large numbers", 100000, 50000, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cost := model.EstimateCost(test.inputTokens, test.outputTokens)

			if test.expectCost && cost < 0 {
				t.Errorf("Expected non-negative cost, got %f", cost)
			}

			// Manual calculation to verify
			expectedCost := (float64(test.inputTokens)/1000000)*model.InputCost +
				(float64(test.outputTokens)/1000000)*model.OutputCost

			if cost != expectedCost {
				t.Errorf("Expected cost %f, got %f", expectedCost, cost)
			}
		})
	}
}

func TestGetCheapestModel(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
	}{
		{"OpenAI", ProviderOpenAI},
		{"Anthropic", ProviderAnthropic},
		{"Unknown", Provider("unknown")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model, err := GetCheapestModel(test.provider)

			if test.provider == Provider("unknown") {
				if err == nil {
					t.Errorf("Expected error for unknown provider, got model %s", model.Name)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected model for provider %s, got error: %v", test.provider, err)
				return
			}

			// Verify the returned model belongs to the provider
			modelInfo := model

			if modelInfo.Provider != test.provider {
				t.Errorf("Returned model %s belongs to %s, expected %s",
					model.Name, modelInfo.Provider, test.provider)
			}
		})
	}
}

func TestGetMostCapableModel(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
	}{
		{"OpenAI", ProviderOpenAI},
		{"Anthropic", ProviderAnthropic},
		{"Unknown", Provider("unknown")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model, err := GetMostCapableModel(test.provider)

			if test.provider == Provider("unknown") {
				if err == nil {
					t.Errorf("Expected error for unknown provider, got model %s", model.Name)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected model for provider %s, got error: %v", test.provider, err)
				return
			}

			// Verify the returned model belongs to the provider
			modelInfo := model

			if modelInfo.Provider != test.provider {
				t.Errorf("Returned model %s belongs to %s, expected %s",
					model.Name, modelInfo.Provider, test.provider)
			}
		})
	}
}

func TestAllModelsHaveValidData(t *testing.T) {
	// Get all model names
	allModels := []string{
		ModelGPT4o, ModelGPT4oMini, ModelGPT4Turbo, ModelGPT35Turbo,
		ModelClaude35Sonnet, ModelClaude35Haiku, ModelClaudeOpus, ModelClaudeSonnet,
	}

	for _, modelName := range allModels {
		t.Run(modelName, func(t *testing.T) {
			model, err := GetModel(modelName)
			if err != nil {
				t.Fatalf("Model %s not found in models map: %v", modelName, err)
			}

			// Validate all required fields
			if model.Name != modelName {
				t.Errorf("Model name mismatch: expected %s, got %s", modelName, model.Name)
			}

			if model.Provider == "" {
				t.Error("Provider is empty")
			}

			if model.ContextSize <= 0 {
				t.Errorf("Invalid context size: %d", model.ContextSize)
			}

			if model.InputCost < 0 {
				t.Errorf("Negative input cost: %f", model.InputCost)
			}

			if model.OutputCost < 0 {
				t.Errorf("Negative output cost: %f", model.OutputCost)
			}

			// Validate provider values
			validProviders := []Provider{ProviderOpenAI, ProviderAnthropic}
			validProvider := false
			for _, vp := range validProviders {
				if model.Provider == vp {
					validProvider = true
					break
				}
			}
			if !validProvider {
				t.Errorf("Invalid provider: %s", model.Provider)
			}
		})
	}
}

func TestModelComparison(t *testing.T) {
	// Test that GPT-4o Mini is cheaper than GPT-4o
	gpt4o, _ := GetModel(ModelGPT4o)
	gpt4oMini, _ := GetModel(ModelGPT4oMini)

	if gpt4oMini.InputCost >= gpt4o.InputCost {
		t.Errorf("GPT-4o Mini should be cheaper than GPT-4o for input tokens: %f vs %f",
			gpt4oMini.InputCost, gpt4o.InputCost)
	}

	// Test that Claude 3.5 Haiku is cheaper than Claude 3.5 Sonnet
	claudeSonnet, _ := GetModel(ModelClaude35Sonnet)
	claudeHaiku, _ := GetModel(ModelClaude35Haiku)

	if claudeHaiku.InputCost >= claudeSonnet.InputCost {
		t.Errorf("Claude 3.5 Haiku should be cheaper than Claude 3.5 Sonnet for input tokens: %f vs %f",
			claudeHaiku.InputCost, claudeSonnet.InputCost)
	}
}
