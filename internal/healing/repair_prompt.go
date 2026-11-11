package healing

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SynthesizeRepairPrompt generates targeted correction instructions without discarding valid prior context.
func SynthesizeRepairPrompt(
	originalPrompt string,
	priorOutput string,
	category ErrorCategory,
	rawError string,
	validationErrors []string,
	schema map[string]any,
	customHint string,
) string {
	var builder strings.Builder

	builder.WriteString(originalPrompt)
	builder.WriteString("\n\n")
	builder.WriteString("=== SYSTEM SELF-HEALING CORRECTION DIRECTIVE ===\n")
	builder.WriteString(fmt.Sprintf("Your previous response failed validation with category [%s].\n", category))

	if len(validationErrors) > 0 {
		builder.WriteString("Exact validation failures detected:\n")
		for i, errStr := range validationErrors {
			builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, errStr))
		}
	} else if rawError != "" {
		builder.WriteString(fmt.Sprintf("Error encountered: %s\n", rawError))
	}

	if customHint != "" {
		builder.WriteString(fmt.Sprintf("\nOperational Guidance: %s\n", customHint))
	}

	if schema != nil {
		schemaBytes, err := json.MarshalIndent(schema, "", "  ")
		if err == nil {
			builder.WriteString("\nYou must strictly conform your response to this JSON schema:\n")
			builder.WriteString(string(schemaBytes))
			builder.WriteString("\n")
		}
	}

	if priorOutput != "" {
		builder.WriteString("\nYour previous raw output was:\n")
		builder.WriteString(priorOutput)
		builder.WriteString("\n\nFix the errors above, retain all verified factual details, and return only the corrected JSON payload.")
	} else {
		builder.WriteString("\nPlease correct the execution and return only the valid JSON payload.")
	}

	return builder.String()
}
