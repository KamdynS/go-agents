package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClientConfig holds MCP server connection details
type ClientConfig struct {
	BaseURL string
	Headers map[string]string
	Timeout time.Duration
}

// Client provides minimal MCP client functionality: list tools and execute
type Client struct {
	cfg    ClientConfig
	client *http.Client
}

func NewClient(cfg ClientConfig) *Client {
	hc := &http.Client{Timeout: cfg.Timeout}
	if cfg.Timeout == 0 {
		hc.Timeout = 15 * time.Second
	}
	return &Client{cfg: cfg, client: hc}
}

type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema"`
}

type listToolsResp struct {
	Tools []ToolInfo `json:"tools"`
}

// ListTools fetches tool metadata from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	url := fmt.Sprintf("%s/tools", c.cfg.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mcp list tools failed: %s: %s", resp.Status, string(b))
	}
	var out listToolsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Tools, nil
}

type execReq struct {
	Input string `json:"input"`
}
type execResp struct {
	Result string `json:"result"`
}

// ExecuteTool runs the named tool with the given input.
func (c *Client) ExecuteTool(ctx context.Context, name string, input string) (string, error) {
	url := fmt.Sprintf("%s/tools/%s/execute", c.cfg.BaseURL, name)
	body, _ := json.Marshal(execReq{Input: input})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mcp execute failed: %s: %s", resp.Status, string(b))
	}
	var out execResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Result, nil
}
