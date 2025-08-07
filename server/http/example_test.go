package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"time"

	"github.com/KamdynS/go-agents/agent/core"
)

type exAgent struct{}

func (exAgent) Run(ctx context.Context, input core.Message) (core.Message, error) {
	return core.Message{Role: "assistant", Content: "pong"}, nil
}
func (exAgent) RunStream(ctx context.Context, input core.Message, output chan<- core.Message) error {
	defer close(output)
	output <- core.Message{Role: "assistant", Content: "pong"}
	return nil
}

func ExampleServer_chat() {
	s := NewServer(exAgent{}, Config{})
	reqBody, _ := json.Marshal(ChatRequest{Message: "ping"})
	req := httptest.NewRequest("POST", "/chat", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.chatHandler(w, req)
	fmt.Println(w.Code)
	// Output:
	// 200
}

func ExampleServer_stream() {
	s := NewServer(exAgent{}, Config{})
	reqBody, _ := json.Marshal(ChatRequest{Message: "ping"})
	req := httptest.NewRequest("POST", "/chat/stream", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.streamHandler(w, req)
	time.Sleep(10 * time.Millisecond)
	fmt.Println(w.Code)
	// Output:
	// 200
}
