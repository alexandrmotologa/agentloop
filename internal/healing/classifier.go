package healing

import (
	"strings"
)

// ErrorCategory defines the classification of a failure.
type ErrorCategory string

const (
	CategorySchemaMismatch   ErrorCategory = "SCHEMA_MISMATCH"
	CategoryJSONSyntax       ErrorCategory = "JSON_SYNTAX_ERROR"
	CategoryLowConfidence    ErrorCategory = "LOW_CONFIDENCE"
	CategoryNetworkTransient ErrorCategory = "NETWORK_TRANSIENT"
	CategoryToolError        ErrorCategory = "TOOL_ERROR"
	CategoryLogicalError     ErrorCategory = "LOGICAL_ERROR"
	CategoryUnknown          ErrorCategory = "UNKNOWN"
)

// Classify determines the root cause category for a node failure.
func Classify(err error, validationErrors []string, confidence float64, threshold float64) ErrorCategory {
	if len(validationErrors) > 0 {
		for _, ve := range validationErrors {
			if strings.Contains(strings.ToLower(ve), "json syntax") || strings.Contains(strings.ToLower(ve), "unexpected end") {
				return CategoryJSONSyntax
			}
		}
		return CategorySchemaMismatch
	}

	if threshold > 0 && confidence < threshold {
		return CategoryLowConfidence
	}

	if err == nil {
		return CategoryUnknown
	}

	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "syntax error") || strings.Contains(errStr, "invalid character") || strings.Contains(errStr, "unmarshal") {
		return CategoryJSONSyntax
	}

	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "reset by peer") {
		return CategoryNetworkTransient
	}

	if strings.Contains(errStr, "tool") || strings.Contains(errStr, "query execution") || strings.Contains(errStr, "sql") {
		return CategoryToolError
	}

	return CategoryLogicalError
}
