package llm

import (
	"context"
)

// GenerateRequest contains parameters for LLM text or structured generation.
type GenerateRequest struct {
	Model        string         `json:"model"`
	Prompt       string         `json:"prompt"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	Temperature  float64        `json:"temperature"`
	MaxTokens    int            `json:"max_tokens,omitempty"`
	OutputSchema map[string]any `json:"output_schema,omitempty"`
	Tools        []ToolDef      `json:"tools,omitempty"`
}

// ToolDef defines a callable tool for function calling.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall represents a model-requested tool invocation.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// GenerateResponse holds the completion output, structured data, and metrics.
type GenerateResponse struct {
	Content          string         `json:"content"`
	StructuredOutput map[string]any `json:"structured_output,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	Confidence       float64        `json:"confidence"`
	TokensConsumed   int            `json:"tokens_consumed"`
}

// Client is the unified interface for interacting with LLM providers.
type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}
