package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientListAndExecute(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(listToolsResp{Tools: []ToolInfo{{Name: "echo", Description: "Echo", Schema: map[string]any{"type": "object"}}}})
	})
	mux.HandleFunc("/tools/echo/execute", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(execResp{Result: "ok"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL, Timeout: time.Second})
	tools, err := c.ListTools(context.Background())
	if err != nil || len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("list: %v %+v", err, tools)
	}
	res, err := c.ExecuteTool(context.Background(), "echo", "hi")
	if err != nil || res != "ok" {
		t.Fatalf("exec: %v %q", err, res)
	}
}

func TestClientHTTPErrorPaths(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("bad"))
	})
	mux.HandleFunc("/tools/x/execute", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("bad exec"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(ClientConfig{BaseURL: srv.URL})
	if _, err := c.ListTools(context.Background()); err == nil {
		t.Fatalf("expected list error")
	}
	if _, err := c.ExecuteTool(context.Background(), "x", ""); err == nil {
		t.Fatalf("expected exec error")
	}
}
