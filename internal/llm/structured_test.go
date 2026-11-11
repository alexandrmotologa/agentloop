package llm

import (
	"testing"
)

func TestParseAndValidate(t *testing.T) {
	schema := map[string]any{
		"type":     "object",
		"required": []any{"discrepancies", "is_approved", "variance_percentage", "summary"},
		"properties": map[string]any{
			"discrepancies":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"is_approved":         map[string]any{"type": "boolean"},
			"variance_percentage": map[string]any{"type": "number"},
			"summary":             map[string]any{"type": "string"},
		},
	}

	content := `{"discrepancies":[],"is_approved":true,"summary":"Invoice totals match approved ERP Purchase Order exactly. No variances found.","variance_percentage":0.0}`

	data, valErrors, err := ParseAndValidate(content, schema)
	if err != nil {
		t.Fatalf("ParseAndValidate failed: %v", err)
	}
	if len(valErrors) > 0 {
		t.Errorf("Unexpected validation errors: %v", valErrors)
	}
	t.Logf("Parsed data: %v", data)

	conf := ComputeConfidence(content, data, valErrors)
	t.Logf("Confidence: %.2f", conf)
	if conf < 0.85 {
		t.Errorf("Expected confidence >= 0.85, got %.2f", conf)
	}
}
