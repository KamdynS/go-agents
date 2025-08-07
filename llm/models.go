package llm

import (
	"fmt"
	"strings"
)

// Model represents an LLM model with its properties
type Model struct {
	Provider    Provider    `json:"provider"`
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Family      ModelFamily `json:"family"`
	ContextSize int         `json:"context_size"`
	InputCost   float64     `json:"input_cost"`  // Cost per 1M input tokens in USD
	OutputCost  float64     `json:"output_cost"` // Cost per 1M output tokens in USD
	Capabilities Capabilities `json:"capabilities"`
}

// Provider represents LLM providers
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

// ModelFamily represents model families/series
type ModelFamily string

const (
	// OpenAI families
	FamilyGPT4o      ModelFamily = "gpt-4o"
	FamilyGPT4       ModelFamily = "gpt-4"
	FamilyGPT35      ModelFamily = "gpt-3.5"
	FamilyO1         ModelFamily = "o1"
	
	// Anthropic families
	FamilyClaude3    ModelFamily = "claude-3"
	FamilyClaude35   ModelFamily = "claude-3.5"
	FamilyClaude4    ModelFamily = "claude-4"
)

// Capabilities represents what a model can do
type Capabilities struct {
	Chat            bool `json:"chat"`
	FunctionCalling bool `json:"function_calling"`
	Vision          bool `json:"vision"`
	Reasoning       bool `json:"reasoning"`
	ToolUse         bool `json:"tool_use"`
	JSON            bool `json:"json"`
	Streaming       bool `json:"streaming"`
}

// OpenAI Models
const (
	// GPT-4o family
	ModelGPT4o     = "gpt-4o"
	ModelGPT4oMini = "gpt-4o-mini"
	
	// GPT-4 family  
	ModelGPT4Turbo         = "gpt-4-turbo"
	ModelGPT4              = "gpt-4"
	ModelGPT4_32k          = "gpt-4-32k"
	ModelGPT4_1106_Preview = "gpt-4-1106-preview"
	
	// GPT-3.5 family
	ModelGPT35Turbo     = "gpt-3.5-turbo"
	ModelGPT35Turbo16k  = "gpt-3.5-turbo-16k"
	
	// O1 reasoning family
	ModelO1        = "o1"
	ModelO1Preview = "o1-preview"
	ModelO1Mini    = "o1-mini"
)

// Anthropic Models
const (
	// Claude 4 family (latest)
	ModelClaudeOpus4   = "claude-4-opus"
	ModelClaudeSonnet4 = "claude-4-sonnet"
	
	// Claude 3.5 family
	ModelClaude35Sonnet = "claude-3-5-sonnet-20241022"
	ModelClaude35Haiku  = "claude-3-5-haiku-20241022"
	
	// Claude 3 family (legacy)
	ModelClaudeOpus   = "claude-3-opus-20240229"
	ModelClaudeSonnet = "claude-3-sonnet-20240229"  
	ModelClaudeHaiku  = "claude-3-haiku-20240307"
)

// AvailableModels contains all available models with their metadata
var AvailableModels = map[string]Model{
	// OpenAI GPT-4o family
	ModelGPT4o: {
		Provider:    ProviderOpenAI,
		Name:        ModelGPT4o,
		DisplayName: "GPT-4o",
		Family:      FamilyGPT4o,
		ContextSize: 128000,
		InputCost:   5.0,   // $5/1M tokens
		OutputCost:  15.0,  // $15/1M tokens
		Capabilities: Capabilities{
			Chat: true, FunctionCalling: true, Vision: true, JSON: true, Streaming: true,
		},
	},
	ModelGPT4oMini: {
		Provider:    ProviderOpenAI,
		Name:        ModelGPT4oMini,
		DisplayName: "GPT-4o Mini",
		Family:      FamilyGPT4o,
		ContextSize: 128000,
		InputCost:   0.15,  // $0.15/1M tokens
		OutputCost:  0.60,  // $0.60/1M tokens
		Capabilities: Capabilities{
			Chat: true, FunctionCalling: true, Vision: true, JSON: true, Streaming: true,
		},
	},
	
	// OpenAI GPT-4 family
	ModelGPT4Turbo: {
		Provider:    ProviderOpenAI,
		Name:        ModelGPT4Turbo,
		DisplayName: "GPT-4 Turbo",
		Family:      FamilyGPT4,
		ContextSize: 128000,
		InputCost:   10.0,  // $10/1M tokens
		OutputCost:  30.0,  // $30/1M tokens
		Capabilities: Capabilities{
			Chat: true, FunctionCalling: true, Vision: true, JSON: true, Streaming: true,
		},
	},
	ModelGPT4: {
		Provider:    ProviderOpenAI,
		Name:        ModelGPT4,
		DisplayName: "GPT-4",
		Family:      FamilyGPT4,
		ContextSize: 8192,
		InputCost:   30.0,  // $30/1M tokens
		OutputCost:  60.0,  // $60/1M tokens
		Capabilities: Capabilities{
			Chat: true, FunctionCalling: true, JSON: true, Streaming: true,
		},
	},
	
	// OpenAI GPT-3.5 family
	ModelGPT35Turbo: {
		Provider:    ProviderOpenAI,
		Name:        ModelGPT35Turbo,
		DisplayName: "GPT-3.5 Turbo",
		Family:      FamilyGPT35,
		ContextSize: 16385,
		InputCost:   0.50,  // $0.50/1M tokens
		OutputCost:  1.50,  // $1.50/1M tokens
		Capabilities: Capabilities{
			Chat: true, FunctionCalling: true, JSON: true, Streaming: true,
		},
	},
	
	// OpenAI O1 reasoning family
	ModelO1Preview: {
		Provider:    ProviderOpenAI,
		Name:        ModelO1Preview,
		DisplayName: "O1 Preview",
		Family:      FamilyO1,
		ContextSize: 128000,
		InputCost:   15.0,  // $15/1M tokens
		OutputCost:  60.0,  // $60/1M tokens
		Capabilities: Capabilities{
			Chat: true, Reasoning: true, JSON: true,
		},
	},
	ModelO1Mini: {
		Provider:    ProviderOpenAI,
		Name:        ModelO1Mini,
		DisplayName: "O1 Mini",
		Family:      FamilyO1,
		ContextSize: 128000,
		InputCost:   3.0,   // $3/1M tokens
		OutputCost:  12.0,  // $12/1M tokens
		Capabilities: Capabilities{
			Chat: true, Reasoning: true, JSON: true,
		},
	},
	
	// Anthropic Claude 4 family
	ModelClaudeOpus4: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaudeOpus4,
		DisplayName: "Claude 4 Opus",
		Family:      FamilyClaude4,
		ContextSize: 200000,
		InputCost:   15.0,  // $15/1M tokens
		OutputCost:  75.0,  // $75/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
	ModelClaudeSonnet4: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaudeSonnet4,
		DisplayName: "Claude 4 Sonnet",
		Family:      FamilyClaude4,
		ContextSize: 200000,
		InputCost:   3.0,   // $3/1M tokens
		OutputCost:  15.0,  // $15/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
	
	// Anthropic Claude 3.5 family
	ModelClaude35Sonnet: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaude35Sonnet,
		DisplayName: "Claude 3.5 Sonnet",
		Family:      FamilyClaude35,
		ContextSize: 200000,
		InputCost:   3.0,   // $3/1M tokens
		OutputCost:  15.0,  // $15/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
	ModelClaude35Haiku: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaude35Haiku,
		DisplayName: "Claude 3.5 Haiku",
		Family:      FamilyClaude35,
		ContextSize: 200000,
		InputCost:   0.25,  // $0.25/1M tokens
		OutputCost:  1.25,  // $1.25/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
	
	// Anthropic Claude 3 family (legacy)
	ModelClaudeOpus: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaudeOpus,
		DisplayName: "Claude 3 Opus",
		Family:      FamilyClaude3,
		ContextSize: 200000,
		InputCost:   15.0,  // $15/1M tokens
		OutputCost:  75.0,  // $75/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
	ModelClaudeSonnet: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaudeSonnet,
		DisplayName: "Claude 3 Sonnet",
		Family:      FamilyClaude3,
		ContextSize: 200000,
		InputCost:   3.0,   // $3/1M tokens
		OutputCost:  15.0,  // $15/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
	ModelClaudeHaiku: {
		Provider:    ProviderAnthropic,
		Name:        ModelClaudeHaiku,
		DisplayName: "Claude 3 Haiku",
		Family:      FamilyClaude3,
		ContextSize: 200000,
		InputCost:   0.25,  // $0.25/1M tokens
		OutputCost:  1.25,  // $1.25/1M tokens
		Capabilities: Capabilities{
			Chat: true, Vision: true, ToolUse: true, JSON: true, Streaming: true,
		},
	},
}

// GetModel returns model metadata for a given model name
func GetModel(name string) (Model, error) {
	model, exists := AvailableModels[name]
	if !exists {
		return Model{}, fmt.Errorf("unknown model: %s", name)
	}
	return model, nil
}

// GetModelsByProvider returns all models for a given provider
func GetModelsByProvider(provider Provider) []Model {
	var models []Model
	for _, model := range AvailableModels {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// GetModelsByFamily returns all models in a given family
func GetModelsByFamily(family ModelFamily) []Model {
	var models []Model
	for _, model := range AvailableModels {
		if model.Family == family {
			models = append(models, model)
		}
	}
	return models
}

// GetCheapestModel returns the cheapest model for a provider
func GetCheapestModel(provider Provider) (Model, error) {
	models := GetModelsByProvider(provider)
	if len(models) == 0 {
		return Model{}, fmt.Errorf("no models found for provider: %s", provider)
	}
	
	cheapest := models[0]
	for _, model := range models {
		totalCost := model.InputCost + model.OutputCost
		cheapestCost := cheapest.InputCost + cheapest.OutputCost
		if totalCost < cheapestCost {
			cheapest = model
		}
	}
	return cheapest, nil
}

// GetMostCapableModel returns the most capable model for a provider
func GetMostCapableModel(provider Provider) (Model, error) {
	models := GetModelsByProvider(provider)
	if len(models) == 0 {
		return Model{}, fmt.Errorf("no models found for provider: %s", provider)
	}
	
	// Simple heuristic: higher context size + more capabilities = more capable
	var mostCapable Model
	maxScore := -1
	
	for _, model := range models {
		score := model.ContextSize / 1000 // Context size in thousands
		
		// Add points for capabilities
		if model.Capabilities.Vision {
			score += 10
		}
		if model.Capabilities.FunctionCalling || model.Capabilities.ToolUse {
			score += 8
		}
		if model.Capabilities.Reasoning {
			score += 12
		}
		
		// Prefer newer families
		if strings.Contains(string(model.Family), "4") {
			score += 20
		} else if strings.Contains(string(model.Family), "3.5") {
			score += 15
		}
		
		if score > maxScore {
			maxScore = score
			mostCapable = model
		}
	}
	
	return mostCapable, nil
}

// ValidateModel checks if a model name is valid
func ValidateModel(name string) error {
	_, err := GetModel(name)
	return err
}

// String returns a human-readable representation of the model
func (m Model) String() string {
	return fmt.Sprintf("%s (%s) - %s", m.DisplayName, m.Name, m.Provider)
}

// EstimateCost estimates the cost for given token counts
func (m Model) EstimateCost(inputTokens, outputTokens int) float64 {
	inputCost := (float64(inputTokens) / 1000000) * m.InputCost
	outputCost := (float64(outputTokens) / 1000000) * m.OutputCost
	return inputCost + outputCost
}