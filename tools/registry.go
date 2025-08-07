package tools

import (
	"context"
	"fmt"
	"sync"
)

// Tool defines the interface for agent tools
type Tool interface {
	// Name returns the tool's name for identification
	Name() string
	
	// Description returns a human-readable description of what the tool does
	Description() string
	
	// Execute runs the tool with the given input and returns the result
	Execute(ctx context.Context, input string) (string, error)
	
	// Schema returns the JSON schema for the tool's input (optional)
	Schema() map[string]interface{}
}

// Registry manages a collection of tools available to agents
type Registry interface {
	// Register adds a tool to the registry
	Register(tool Tool) error
	
	// Get retrieves a tool by name
	Get(name string) (Tool, bool)
	
	// List returns all available tool names
	List() []string
	
	// Execute runs a tool by name with the given input
	Execute(ctx context.Context, name string, input string) (string, error)
}

// DefaultRegistry is a simple in-memory tool registry
type DefaultRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new DefaultRegistry
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		tools: make(map[string]Tool),
	}
}

// Register implements Registry interface
func (r *DefaultRegistry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	
	r.tools[name] = tool
	return nil
}

// Get implements Registry interface
func (r *DefaultRegistry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, exists := r.tools[name]
	return tool, exists
}

// List implements Registry interface
func (r *DefaultRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Execute implements Registry interface
func (r *DefaultRegistry) Execute(ctx context.Context, name string, input string) (string, error) {
	tool, exists := r.Get(name)
	if !exists {
		return "", fmt.Errorf("tool %s not found", name)
	}
	
	return tool.Execute(ctx, input)
}