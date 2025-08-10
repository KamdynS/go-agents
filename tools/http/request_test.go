package http

import (
	"context"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestToolGetAndPost(t *testing.T) {
	srv := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method == stdhttp.MethodGet {
			w.WriteHeader(200)
			_, _ = w.Write([]byte("hello"))
			return
		}
		if r.Method == stdhttp.MethodPost {
			w.WriteHeader(201)
			_, _ = w.Write([]byte("created"))
			return
		}
		w.WriteHeader(405)
	}))
	defer srv.Close()

	tool := NewRequestTool(0)
	out, err := tool.Execute(context.Background(), "GET|"+srv.URL+"|")
	if err != nil || out == "" {
		t.Fatalf("get failed: %v %q", err, out)
	}
	out, err = tool.Execute(context.Background(), "POST|"+srv.URL+"|{\"a\":1}")
	if err != nil || out == "" {
		t.Fatalf("post failed: %v %q", err, out)
	}
}

func TestRequestToolBadInput(t *testing.T) {
	tool := NewRequestTool(0)
	if _, err := tool.Execute(context.Background(), "BAD"); err == nil {
		t.Fatalf("expected input error")
	}
}
