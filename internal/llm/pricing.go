package llm

import (
	"strings"
)

// EstimateCost calculates estimated expenditure in USD for consumed tokens.
func EstimateCost(model string, tokens int) float64 {
	if tokens <= 0 {
		return 0.0
	}

	m := strings.ToLower(model)

	var ratePerThousand float64

	switch {
	case strings.Contains(m, "gpt-4o-mini"):
		ratePerThousand = 0.000375
	case strings.Contains(m, "gpt-4o"):
		ratePerThousand = 0.00625
	case strings.Contains(m, "claude-3-5-sonnet"):
		ratePerThousand = 0.009
	case strings.Contains(m, "claude-3-haiku"):
		ratePerThousand = 0.00075
	case strings.Contains(m, "ollama") || strings.Contains(m, "mock") || strings.Contains(m, "local"):
		ratePerThousand = 0.0
	default:
		ratePerThousand = 0.002
	}

	cost := (float64(tokens) / 1000.0) * ratePerThousand
	return cost
}
