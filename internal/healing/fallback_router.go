package healing

import (
	"fmt"
)

// SelectFallbackTool verifies and returns the configured fallback tool name.
func SelectFallbackTool(fallbackTool string, params map[string]any) (string, map[string]any, error) {
	if fallbackTool == "" {
		return "", nil, fmt.Errorf("no fallback tool configured")
	}

	return fallbackTool, params, nil
}
