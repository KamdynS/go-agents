package llm

import (
	"context"
	"fmt"
)

type fakeClient struct{}

func (f *fakeClient) Chat(ctx context.Context, req *ChatRequest) (*Response, error) {
	return &Response{Content: "ok", Model: "test", Provider: ProviderOpenAI}, nil
}
func (f *fakeClient) Completion(ctx context.Context, prompt string) (*Response, error) {
	return &Response{Content: "ok", Model: "test", Provider: ProviderOpenAI}, nil
}
func (f *fakeClient) Stream(ctx context.Context, req *ChatRequest, output chan<- *Response) error {
	defer close(output)
	output <- &Response{Content: "ok", Model: "test", Provider: ProviderOpenAI}
	return nil
}
func (f *fakeClient) Model() string      { return "test" }
func (f *fakeClient) Provider() Provider { return ProviderOpenAI }
func (f *fakeClient) Validate() error    { return nil }

func ExampleInstrumentedClient() {
	c := NewInstrumentedClient(&fakeClient{})
	r, _ := c.Chat(context.Background(), &ChatRequest{Messages: []Message{{Role: "user", Content: "hi"}}})
	fmt.Println(r.Content)
	// Output:
	// ok
}
