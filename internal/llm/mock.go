package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// MockClient provides deterministic simulated LLM responses for testing and demonstrations.
type MockClient struct {
	mu            sync.Mutex
	CallCount     int
	Responses     []*GenerateResponse
	ResponderFunc func(req GenerateRequest, callCount int) (*GenerateResponse, error)
}

// NewMockClient creates a mock LLM client with optional queued responses.
func NewMockClient(responses ...*GenerateResponse) *MockClient {
	return &MockClient{
		Responses: responses,
	}
}

func (m *MockClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++

	if m.ResponderFunc != nil {
		return m.ResponderFunc(req, m.CallCount)
	}

	if len(m.Responses) > 0 {
		idx := (m.CallCount - 1) % len(m.Responses)
		resp := m.Responses[idx]
		return resp, nil
	}

	// Default intelligent generator based on prompt and schema
	return m.defaultGenerate(req)
}

func (m *MockClient) defaultGenerate(req GenerateRequest) (*GenerateResponse, error) {
	// If repairing from a previous error or containing repair keywords
	isRepairPrompt := strings.Contains(strings.ToLower(req.Prompt), "repair") ||
		strings.Contains(strings.ToLower(req.Prompt), "validation error") ||
		strings.Contains(strings.ToLower(req.Prompt), "correct the json")

	// If prompt is for reconciliation & audit
	if strings.Contains(strings.ToLower(req.Prompt), "reconcile") ||
		strings.Contains(strings.ToLower(req.Prompt), "audit") ||
		strings.Contains(strings.ToLower(req.Prompt), "variance") ||
		strings.Contains(strings.ToLower(req.Prompt), "discrepancies") ||
		(req.OutputSchema != nil && req.OutputSchema["properties"] != nil && req.OutputSchema["properties"].(map[string]any)["variance_percentage"] != nil) {
		output := map[string]any{
			"discrepancies":       []any{},
			"is_approved":         true,
			"variance_percentage": 0.0,
			"summary":             "Invoice totals match approved ERP Purchase Order exactly. No variances found.",
		}
		bytes, _ := json.Marshal(output)
		return &GenerateResponse{
			Content:          string(bytes),
			StructuredOutput: output,
			Confidence:       0.98,
			TokensConsumed:   195,
		}, nil
	}

	// If prompt is for invoice extraction
	if strings.Contains(strings.ToLower(req.Prompt), "invoice") ||
		(req.OutputSchema != nil && req.OutputSchema["properties"] != nil && req.OutputSchema["properties"].(map[string]any)["line_items"] != nil) {
		output := map[string]any{
			"vendor_name":  "Acme Industrial Solutions Inc.",
			"tax_id":       "US-94827104",
			"total_amount": 4250.00,
			"currency":     "USD",
			"line_items": []any{
				map[string]any{
					"description": "Enterprise Server Rack Units",
					"quantity":    2,
					"unit_price":  1800.00,
					"total":       3600.00,
				},
				map[string]any{
					"description": "Gigabit Network Switches",
					"quantity":    1,
					"unit_price":  650.00,
					"total":       650.00,
				},
			},
		}

		bytes, _ := json.Marshal(output)
		return &GenerateResponse{
			Content:          string(bytes),
			StructuredOutput: output,
			Confidence:       0.96,
			TokensConsumed:   284,
		}, nil
	}

	// Generic fallback mock response
	content := fmt.Sprintf(`{"status": "ok", "message": "mock response for call %d", "repaired": %v}`, m.CallCount, isRepairPrompt)
	var structured map[string]any
	_ = json.Unmarshal([]byte(content), &structured)

	return &GenerateResponse{
		Content:          content,
		StructuredOutput: structured,
		Confidence:       0.92,
		TokensConsumed:   120,
	}, nil
}
