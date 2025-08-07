package llm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// Structured represents a type that can be used for structured output
type Structured interface {
	// Validate validates the structured output
	Validate() error
	// JSONSchema returns the JSON schema for this type
	JSONSchema() map[string]interface{}
}

// StructuredRequest wraps a request with structured output requirements
type StructuredRequest[T Structured] struct {
	Messages     []Message              `json:"messages"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	Model        string                 `json:"model"`
	Temperature  float64                `json:"temperature,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Schema       map[string]interface{} `json:"schema,omitempty"`
	OutputType   T                      `json:"-"` // Template for the output type
}

// StructuredResponse contains the parsed and validated structured output
type StructuredResponse[T Structured] struct {
	Data        T                 `json:"data"`
	RawResponse *Response         `json:"raw_response"`
	Usage       *Usage            `json:"usage,omitempty"`
	Validation  *ValidationResult `json:"validation,omitempty"`
}

// ValidationResult contains details about validation
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Retries int      `json:"retries"`
	RawJSON string   `json:"raw_json,omitempty"`
}

// Usage contains token usage information
type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalTokens  int     `json:"total_tokens"`
	Cost         float64 `json:"cost,omitempty"`
}

// StructuredClient extends the base Client with structured output capabilities
// Note: Generic methods are implemented as standalone functions due to Go language limitations
type StructuredClient interface {
	Client
}

// BaseStructured provides common functionality for structured types
type BaseStructured struct{}

// Validate implements basic validation (override in specific types)
func (b BaseStructured) Validate() error {
	return nil
}

// JSONSchema generates a basic JSON schema from struct tags
func (b BaseStructured) JSONSchema() map[string]interface{} {
	return generateJSONSchema(b)
}

// Common structured output types

// TextClassification represents text classification output
type TextClassification struct {
	BaseStructured
	Label      string  `json:"label" description:"The predicted label/category"`
	Confidence float64 `json:"confidence" description:"Confidence score between 0 and 1"`
	Reasoning  string  `json:"reasoning,omitempty" description:"Explanation of the classification"`
}

func (tc TextClassification) Validate() error {
	if tc.Label == "" {
		return fmt.Errorf("label cannot be empty")
	}
	if tc.Confidence < 0 || tc.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", tc.Confidence)
	}
	return nil
}

// Sentiment represents sentiment analysis output
type Sentiment struct {
	BaseStructured
	Sentiment string  `json:"sentiment" description:"positive, negative, or neutral"`
	Score     float64 `json:"score" description:"Sentiment score between -1 (negative) and 1 (positive)"`
	Magnitude float64 `json:"magnitude,omitempty" description:"Overall strength of emotion"`
}

func (s Sentiment) Validate() error {
	validSentiments := []string{"positive", "negative", "neutral"}
	for _, valid := range validSentiments {
		if s.Sentiment == valid {
			if s.Score < -1 || s.Score > 1 {
				return fmt.Errorf("score must be between -1 and 1, got %f", s.Score)
			}
			return nil
		}
	}
	return fmt.Errorf("sentiment must be one of %v, got %s", validSentiments, s.Sentiment)
}

// JSONSchema overrides the base schema to enforce enum and numeric bounds
func (s Sentiment) JSONSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"sentiment": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"positive", "negative", "neutral"},
				"description": "positive, negative, or neutral",
			},
			"score": map[string]interface{}{
				"type":        "number",
				"minimum":     -1,
				"maximum":     1,
				"description": "Sentiment score between -1 and 1",
			},
			"magnitude": map[string]interface{}{
				"type":        "number",
				"description": "Overall strength of emotion",
			},
		},
		"required": []string{"sentiment", "score"},
	}
}

// Person represents a person entity
type Person struct {
	BaseStructured
	Name     string `json:"name" description:"Full name of the person"`
	Age      int    `json:"age,omitempty" description:"Age in years"`
	Email    string `json:"email,omitempty" description:"Email address"`
	Location string `json:"location,omitempty" description:"Geographic location"`
}

func (p Person) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if p.Age < 0 || p.Age > 150 {
		return fmt.Errorf("age must be between 0 and 150, got %d", p.Age)
	}
	if p.Email != "" && !strings.Contains(p.Email, "@") {
		return fmt.Errorf("invalid email format: %s", p.Email)
	}
	return nil
}

// KeyValueExtraction represents key-value pairs extracted from text
type KeyValueExtraction struct {
	BaseStructured
	Pairs []KeyValue `json:"pairs" description:"Extracted key-value pairs"`
}

type KeyValue struct {
	Key        string      `json:"key" description:"The key or field name"`
	Value      interface{} `json:"value" description:"The extracted value"`
	Confidence float64     `json:"confidence,omitempty" description:"Extraction confidence"`
	Type       string      `json:"type,omitempty" description:"Data type of the value"`
}

func (kve KeyValueExtraction) Validate() error {
	if len(kve.Pairs) == 0 {
		return fmt.Errorf("at least one key-value pair must be extracted")
	}
	for i, pair := range kve.Pairs {
		if pair.Key == "" {
			return fmt.Errorf("key cannot be empty at index %d", i)
		}
		if pair.Value == nil {
			return fmt.Errorf("value cannot be nil for key %s", pair.Key)
		}
	}
	return nil
}

// Summary represents a text summary
type Summary struct {
	BaseStructured
	Title     string   `json:"title" description:"Summary title or headline"`
	Summary   string   `json:"summary" description:"Main summary text"`
	KeyPoints []string `json:"key_points,omitempty" description:"List of key points"`
	WordCount int      `json:"word_count,omitempty" description:"Word count of summary"`
}

func (s Summary) Validate() error {
	if s.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if s.Summary == "" {
		return fmt.Errorf("summary cannot be empty")
	}
	if s.WordCount > 0 && s.WordCount != len(strings.Fields(s.Summary)) {
		return fmt.Errorf("word count mismatch: expected %d, got %d",
			len(strings.Fields(s.Summary)), s.WordCount)
	}
	return nil
}

// QAPair represents a question-answer pair
type QAPair struct {
	BaseStructured
	Question string  `json:"question" description:"The question"`
	Answer   string  `json:"answer" description:"The answer"`
	Context  string  `json:"context,omitempty" description:"Relevant context"`
	Score    float64 `json:"score,omitempty" description:"Confidence score"`
}

func (qa QAPair) Validate() error {
	if qa.Question == "" {
		return fmt.Errorf("question cannot be empty")
	}
	if qa.Answer == "" {
		return fmt.Errorf("answer cannot be empty")
	}
	if qa.Score < 0 || qa.Score > 1 {
		return fmt.Errorf("score must be between 0 and 1, got %f", qa.Score)
	}
	return nil
}

// generateJSONSchema creates a JSON schema from a struct using reflection
func generateJSONSchema(v interface{}) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	if val.Kind() != reflect.Struct {
		return schema
	}

	properties := schema["properties"].(map[string]interface{})
	var required []string

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields and embedded BaseStructured
		if !field.IsExported() || field.Name == "BaseStructured" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// Parse json tag
		jsonName := field.Name
		omitEmpty := false
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				jsonName = parts[0]
			}
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitEmpty = true
				}
			}
		}

		// Generate field schema
		fieldSchema := generateFieldSchema(fieldVal.Type(), field.Tag.Get("description"))
		properties[jsonName] = fieldSchema

		// Add to required if not omitempty
		if !omitEmpty {
			required = append(required, jsonName)
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// generateFieldSchema generates schema for a specific field type
func generateFieldSchema(t reflect.Type, description string) map[string]interface{} {
	schema := make(map[string]interface{})

	if description != "" {
		schema["description"] = description
	}

	switch t.Kind() {
	case reflect.String:
		schema["type"] = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema["type"] = "integer"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema["type"] = "integer"
		schema["minimum"] = 0
	case reflect.Float32, reflect.Float64:
		schema["type"] = "number"
	case reflect.Bool:
		schema["type"] = "boolean"
	case reflect.Slice, reflect.Array:
		schema["type"] = "array"
		schema["items"] = generateFieldSchema(t.Elem(), "")
	case reflect.Map:
		schema["type"] = "object"
	case reflect.Struct:
		if t.Name() != "" {
			// For named structs, generate nested object schema
			schema["type"] = "object"
			// Could recursively generate schema here if needed
		} else {
			schema["type"] = "object"
		}
	case reflect.Interface:
		// For interface{} types, allow any type
		schema["oneOf"] = []map[string]interface{}{
			{"type": "string"},
			{"type": "number"},
			{"type": "boolean"},
			{"type": "object"},
			{"type": "array"},
			{"type": "null"},
		}
	case reflect.Ptr:
		// For pointer types, generate schema for the pointed-to type
		return generateFieldSchema(t.Elem(), description)
	default:
		schema["type"] = "string" // Default fallback
	}

	return schema
}

// ParseStructured attempts to parse JSON into a structured type with validation
func ParseStructured[T Structured](jsonStr string, template T) (*StructuredResponse[T], error) {
	var result T

	// Detect whether T is a pointer type based on the provided template
	templateType := reflect.TypeOf(template)
	wantPtr := templateType.Kind() == reflect.Ptr
	if wantPtr {
		templateType = templateType.Elem()
	}

	// Always unmarshal into a pointer to the underlying struct type
	ptrValue := reflect.New(templateType) // *Underlying

	// Parse JSON into the pointer
	if err := json.Unmarshal([]byte(jsonStr), ptrValue.Interface()); err != nil {
		return nil, fmt.Errorf("json parsing error: %w", err)
	}

	// Convert to the requested generic type T (pointer or value)
	if wantPtr {
		result = ptrValue.Interface().(T) // T is *Underlying
	} else {
		result = ptrValue.Elem().Interface().(T) // T is Underlying
	}

	// Validate
	validationResult := &ValidationResult{
		RawJSON: jsonStr,
	}

	if err := result.Validate(); err != nil {
		validationResult.Valid = false
		validationResult.Errors = []string{err.Error()}
		return &StructuredResponse[T]{
			Data:       result,
			Validation: validationResult,
		}, fmt.Errorf("validation failed: %w", err)
	}

	validationResult.Valid = true

	return &StructuredResponse[T]{
		Data:       result,
		Validation: validationResult,
	}, nil
}
