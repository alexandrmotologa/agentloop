package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAIClient implements Client for OpenAI and OpenAI-compatible endpoints.
type OpenAIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIClient creates an OpenAI API client.
func NewOpenAIClient(apiKey string, baseURL string) *OpenAIClient {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	Temperature    float64             `json:"temperature"`
	MaxTokens      int                 `json:"max_tokens,omitempty"`
	ResponseFormat any                 `json:"response_format,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("openai api key is not set")
	}

	messages := []openAIChatMessage{}
	if req.SystemPrompt != "" {
		messages = append(messages, openAIChatMessage{Role: "system", Content: req.SystemPrompt})
	}
	messages = append(messages, openAIChatMessage{Role: "user", Content: req.Prompt})

	chatReq := openAIChatRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if req.OutputSchema != nil {
		chatReq.ResponseFormat = map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "workflow_structured_output",
				"strict": true,
				"schema": req.OutputSchema,
			},
		}
	}

	bodyBytes, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read openai response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal openai response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("openai returned error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned empty choices")
	}

	content := chatResp.Choices[0].Message.Content
	tokens := chatResp.Usage.TotalTokens

	var structuredData map[string]any
	var valErrors []string
	if req.OutputSchema != nil {
		structuredData, valErrors, _ = ParseAndValidate(content, req.OutputSchema)
	}

	confidence := ComputeConfidence(content, structuredData, valErrors)

	return &GenerateResponse{
		Content:          content,
		StructuredOutput: structuredData,
		Confidence:       confidence,
		TokensConsumed:   tokens,
	}, nil
}
