package tools

import (
	"context"
	"fmt"
	"sync"
)

// ToolFunc defines the execution signature for a registered deterministic tool.
type ToolFunc func(ctx context.Context, params map[string]any) (any, error)

// Registry manages available tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]ToolFunc
}

// NewRegistry creates a new Tool Registry with standard enterprise tools pre-registered.
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]ToolFunc),
	}

	r.Register("http_request", ExecuteHTTPRequest)
	r.Register("sqlite_query", ExecuteSQLiteQuery)
	r.Register("json_transform", ExecuteJSONTransform)
	r.Register("file_read", ExecuteFileRead)
	r.Register("file_write", ExecuteFileWrite)

	return r
}

// Register adds or replaces a tool handler.
func (r *Registry) Register(name string, handler ToolFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = handler
}

// Execute runs a named tool with provided parameters.
func (r *Registry) Execute(ctx context.Context, name string, params map[string]any) (any, error) {
	r.mu.RLock()
	handler, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool '%s' is not registered", name)
	}

	return handler(ctx, params)
}
