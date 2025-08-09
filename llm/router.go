package llm

import (
	"context"
	"errors"
)

// RoutePolicy decides which client/model to use for a given request
type RoutePolicy interface {
	// Select returns the target client to use and (optionally) model override
	Select(req *ChatRequest) (Client, string, error)
}

// StaticPolicy routes by req.Model if present, otherwise uses default
type StaticPolicy struct {
	Default Client
	// Optional explicit model->client map
	ByModel map[string]Client
}

func (p StaticPolicy) Select(req *ChatRequest) (Client, string, error) {
	if req != nil && req.Model != "" {
		if c, ok := p.ByModel[req.Model]; ok && c != nil {
			return c, req.Model, nil
		}
		if p.Default != nil {
			return p.Default, req.Model, nil
		}
		return nil, "", errors.New("no default client configured")
	}
	if p.Default == nil {
		return nil, "", errors.New("no default client configured")
	}
	return p.Default, "", nil
}

// RouterClient implements Client and delegates to inner clients via RoutePolicy
type RouterClient struct {
	policy RoutePolicy
}

func NewRouterClient(policy RoutePolicy) *RouterClient { return &RouterClient{policy: policy} }

func (r *RouterClient) Chat(ctx context.Context, req *ChatRequest) (*Response, error) {
	c, modelOverride, err := r.policy.Select(req)
	if err != nil {
		return nil, err
	}
	if modelOverride != "" {
		req = cloneChatRequest(req)
		req.Model = modelOverride
	}
	return c.Chat(ctx, req)
}

func (r *RouterClient) Completion(ctx context.Context, prompt string) (*Response, error) {
	c, _, err := r.policy.Select(&ChatRequest{})
	if err != nil {
		return nil, err
	}
	return c.Completion(ctx, prompt)
}

func (r *RouterClient) Stream(ctx context.Context, req *ChatRequest, output chan<- *Response) error {
	c, modelOverride, err := r.policy.Select(req)
	if err != nil {
		return err
	}
	if modelOverride != "" {
		req = cloneChatRequest(req)
		req.Model = modelOverride
	}
	return c.Stream(ctx, req, output)
}

func (r *RouterClient) Model() string      { return "router" }
func (r *RouterClient) Provider() Provider { return Provider("router") }
func (r *RouterClient) Validate() error {
	if r.policy == nil {
		return errors.New("nil route policy")
	}
	return nil
}

func cloneChatRequest(req *ChatRequest) *ChatRequest {
	if req == nil {
		return &ChatRequest{}
	}
	cp := *req
	// Shallow copy is fine for our usage
	return &cp
}
