package supervisor

import (
	"context"
	"sync"

	core "github.com/KamdynS/go-agents/agent/core"
)

// Policy defines how a supervisor coordinates agents.
type Policy interface {
	Execute(ctx context.Context, prompt string, agents []core.Agent) (string, error)
}

// SequentialPolicy calls agents one by one, feeding previous output to next.
type SequentialPolicy struct{}

func (SequentialPolicy) Execute(ctx context.Context, prompt string, agents []core.Agent) (string, error) {
	input := prompt
	for _, a := range agents {
		out, err := a.Run(ctx, core.Message{Role: "user", Content: input})
		if err != nil {
			return "", err
		}
		input = out.Content
	}
	return input, nil
}

// FanOutFirst wins: run all agents in parallel and return the first response.
type FanOutFirst struct{}

func (FanOutFirst) Execute(ctx context.Context, prompt string, agents []core.Agent) (string, error) {
	type res struct {
		s   string
		err error
	}
	ch := make(chan res, len(agents))
	var wg sync.WaitGroup
	for _, a := range agents {
		wg.Add(1)
		go func(ag core.Agent) {
			defer wg.Done()
			out, err := ag.Run(ctx, core.Message{Role: "user", Content: prompt})
			if err != nil {
				ch <- res{"", err}
				return
			}
			ch <- res{out.Content, nil}
		}(a)
	}
	// Return first successful; if all fail, return last error
	var lastErr error
	for i := 0; i < len(agents); i++ {
		r := <-ch
		if r.err == nil {
			return r.s, nil
		}
		lastErr = r.err
	}
	wg.Wait()
	return "", lastErr
}
