package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/alexandrmotologa/agentloop/internal/healing"
)

func TestErrorClassifier(t *testing.T) {
	// 1. Schema mismatch
	cat1 := healing.Classify(nil, []string{"missing property 'tax_id'"}, 0.9, 0.8)
	if cat1 != healing.CategorySchemaMismatch {
		t.Errorf("Expected CategorySchemaMismatch, got %s", cat1)
	}

	// 2. Low confidence
	cat2 := healing.Classify(nil, nil, 0.65, 0.85)
	if cat2 != healing.CategoryLowConfidence {
		t.Errorf("Expected CategoryLowConfidence, got %s", cat2)
	}

	// 3. Network transient
	cat3 := healing.Classify(errors.New("HTTP 502 Bad Gateway: upstream timeout"), nil, 1.0, 0.0)
	if cat3 != healing.CategoryNetworkTransient {
		t.Errorf("Expected CategoryNetworkTransient, got %s", cat3)
	}

	// 4. Tool error
	cat4 := healing.Classify(errors.New("sql query execution error: table not found"), nil, 1.0, 0.0)
	if cat4 != healing.CategoryToolError {
		t.Errorf("Expected CategoryToolError, got %s", cat4)
	}
}

func TestSynthesizeRepairPrompt(t *testing.T) {
	originalPrompt := "Extract invoice line items from PDF."
	priorOutput := `{"vendor": "Acme"}`
	valErrors := []string{"missing required property: total_amount"}
	schema := map[string]any{
		"type":     "object",
		"required": []string{"vendor", "total_amount"},
	}

	prompt := healing.SynthesizeRepairPrompt(
		originalPrompt,
		priorOutput,
		healing.CategorySchemaMismatch,
		"",
		valErrors,
		schema,
		"Check total amount at footer",
	)

	if !strings.Contains(prompt, "SYSTEM SELF-HEALING CORRECTION DIRECTIVE") {
		t.Errorf("Expected repair directive header in mutated prompt")
	}
	if !strings.Contains(prompt, "missing required property: total_amount") {
		t.Errorf("Expected validation failure description in mutated prompt")
	}
	if !strings.Contains(prompt, "Check total amount at footer") {
		t.Errorf("Expected operational guidance hint in mutated prompt")
	}
	if !strings.Contains(prompt, `{"vendor": "Acme"}`) {
		t.Errorf("Expected prior raw output preserved for context in mutated prompt")
	}
}
