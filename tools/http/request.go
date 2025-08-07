package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/KamdynS/go-agents/tools"
)

// RequestTool implements a tool for making HTTP requests
type RequestTool struct {
	client  *http.Client
	timeout time.Duration
}

// NewRequestTool creates a new HTTP request tool
func NewRequestTool(timeout time.Duration) *RequestTool {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	
	return &RequestTool{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Name implements tools.Tool interface
func (t *RequestTool) Name() string {
	return "http_request"
}

// Description implements tools.Tool interface
func (t *RequestTool) Description() string {
	return "Makes HTTP requests to external APIs. Input should be in format: METHOD|URL|BODY (optional)"
}

// Execute implements tools.Tool interface
func (t *RequestTool) Execute(ctx context.Context, input string) (string, error) {
	parts := strings.SplitN(input, "|", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid input format. Expected: METHOD|URL|BODY (optional)")
	}
	
	method := strings.ToUpper(parts[0])
	url := parts[1]
	var body string
	if len(parts) > 2 {
		body = parts[2]
	}
	
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set default headers
	if method == "POST" || method == "PUT" || method == "PATCH" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "go-agents/1.0")
	
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	
	result := fmt.Sprintf("Status: %d %s\nBody: %s", 
		resp.StatusCode, resp.Status, string(respBody))
	
	return result, nil
}

// Schema implements tools.Tool interface
func (t *RequestTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "HTTP request in format: METHOD|URL|BODY (optional)",
				"example":     "GET|https://api.example.com/data|",
			},
		},
		"required": []string{"input"},
	}
}

// Ensure RequestTool implements tools.Tool interface
var _ tools.Tool = (*RequestTool)(nil)