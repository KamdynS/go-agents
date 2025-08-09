package core

import (
	"context"
	"errors"
	"strings"

	"github.com/KamdynS/go-agents/llm"
)

// SimpleGuardrails provides minimal input/output filtering and allow/deny checks.
type SimpleGuardrails struct {
	// Deny if any of these substrings appear in the user input
	DenySubstrings []string
	// Allow only if at least one of these substrings appears; if empty, allow all
	AllowSubstrings []string
	// Max input length
	MaxInputChars int
}

func (g *SimpleGuardrails) BeforeLLMCall(ctx context.Context, req *llm.ChatRequest) error {
	if req == nil {
		return nil
	}
	// Enforce input length on last user message
	if len(req.Messages) > 0 {
		last := &req.Messages[len(req.Messages)-1]
		if last.Role == "user" {
			if g.MaxInputChars > 0 && len(last.Content) > g.MaxInputChars {
				last.Content = last.Content[:g.MaxInputChars]
			}
			// Deny substrings
			for _, s := range g.DenySubstrings {
				if s == "" {
					continue
				}
				if strings.Contains(strings.ToLower(last.Content), strings.ToLower(s)) {
					return errors.New("request blocked by guardrails")
				}
			}
			// Allow list
			if len(g.AllowSubstrings) > 0 {
				allowed := false
				for _, s := range g.AllowSubstrings {
					if s == "" {
						continue
					}
					if strings.Contains(strings.ToLower(last.Content), strings.ToLower(s)) {
						allowed = true
						break
					}
				}
				if !allowed {
					return errors.New("request not permitted by guardrails")
				}
			}
		}
	}
	return nil
}

func (g *SimpleGuardrails) AfterLLMResponse(ctx context.Context, resp *llm.Response) error {
	return nil
}
func (g *SimpleGuardrails) BeforeToolExecute(ctx context.Context, toolName string, input string) error {
	return nil
}
func (g *SimpleGuardrails) AfterToolExecute(ctx context.Context, toolName string, result string, execErr error) error {
	return nil
}
func (g *SimpleGuardrails) AfterRun(ctx context.Context, final Message) error { return nil }
