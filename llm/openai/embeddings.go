package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type embeddingsRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

// Embed generates an embedding vector for the given input text using the specified model.
// If model is empty, a reasonable default is used.
func (c *Client) Embed(ctx context.Context, input string, model string) ([]float64, error) {
	if model == "" {
		model = "text-embedding-3-small"
	}
	body, _ := json.Marshal(embeddingsRequest{Input: input, Model: model})

	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	if c.config.Organization != "" {
		req.Header.Set("OpenAI-Organization", c.config.Organization)
	}

	httpClient := &http.Client{Timeout: c.config.Timeout}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var r embeddingsResponse
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode embeddings: %w", err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("openai error: %s", r.Error.Message)
	}
	if len(r.Data) == 0 || len(r.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return r.Data[0].Embedding, nil
}
