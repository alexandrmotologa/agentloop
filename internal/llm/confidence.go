package llm

import (
	"math"
	"strings"
)

var hesitationIndicators = []string{
	"i am not sure",
	"unclear from the document",
	"cannot determine",
	"possibly",
	"may be inaccurate",
	"ambiguous",
	"hard to read",
	"low confidence",
	"ocr artifact",
}

// ComputeConfidence calculates a 0.0 to 1.0 confidence score for an LLM response.
func ComputeConfidence(rawContent string, parsedData map[string]any, validationErrors []string) float64 {
	// If schema validation failed, confidence drops immediately
	if len(validationErrors) > 0 {
		penalty := float64(len(validationErrors)) * 0.25
		return math.Max(0.1, 0.7-penalty)
	}

	// Check if the model provided an explicit confidence field in structured data
	if parsedData != nil {
		if confVal, ok := parsedData["confidence"]; ok {
			switch c := confVal.(type) {
			case float64:
				if c >= 0.0 && c <= 1.0 {
					return c
				}
				if c > 1.0 && c <= 100.0 {
					return c / 100.0
				}
			case int:
				if c >= 0 && c <= 1 {
					return float64(c)
				}
				if c > 1 && c <= 100 {
					return float64(c) / 100.0
				}
			}
		}
	}

	// Analyze raw content for semantic uncertainty signals
	lower := strings.ToLower(rawContent)
	score := 0.95
	for _, phrase := range hesitationIndicators {
		if strings.Contains(lower, phrase) {
			score -= 0.15
		}
	}

	if score < 0.1 {
		score = 0.1
	}
	return math.Round(score*100) / 100
}
