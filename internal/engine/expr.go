package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
)

var templateVarRegex = regexp.MustCompile(`\{\{\s*(\.?state\.[a-zA-Z0-9_\.]+)\s*\}\}`)

// EvaluateCondition evaluates a boolean expression string against the current state.
func EvaluateCondition(condition string, state *State) (bool, error) {
	cond := strings.TrimSpace(condition)
	if cond == "" || cond == "default" || cond == "true" {
		return true, nil
	}
	if cond == "false" {
		return false, nil
	}

	// Normalize syntax: .state.field -> state.field
	normCond := strings.ReplaceAll(cond, ".state.", "state.")
	normCond = strings.TrimPrefix(normCond, ".state.")

	stateMap := state.ToMap()
	env := map[string]any{
		"state": stateMap,
	}

	program, err := expr.Compile(normCond, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("failed to compile condition '%s': %w", condition, err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate condition '%s': %w", condition, err)
	}

	boolVal, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("condition '%s' did not return a boolean", condition)
	}

	return boolVal, nil
}

// Interpolate replaces template variables in the format {{ .state.key }} or {{ state.key }} with state values.
func Interpolate(text string, state *State) (string, error) {
	if !strings.Contains(text, "{{") {
		return text, nil
	}

	stateMap := state.ToMap()
	data := map[string]any{
		"state": stateMap,
	}

	tmpl, err := template.New("prompt").Option("missingkey=zero").Parse(text)
	if err != nil {
		// Fallback to regex substitution if template parse fails
		return fallbackInterpolate(text, state), nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fallbackInterpolate(text, state), nil
	}

	return buf.String(), nil
}

func fallbackInterpolate(text string, state *State) string {
	return templateVarRegex.ReplaceAllStringFunc(text, func(match string) string {
		submatch := templateVarRegex.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		path := submatch[1]
		val, exists := state.Get(path)
		if !exists {
			return ""
		}
		switch v := val.(type) {
		case string:
			return v
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Sprintf("%v", v)
			}
			return string(bytes)
		}
	})
}

// InterpolateValue recursively interpolates strings inside maps, arrays, or primitive values.
func InterpolateValue(val any, state *State) (any, error) {
	if val == nil {
		return nil, nil
	}

	switch v := val.(type) {
	case string:
		return Interpolate(v, state)
	case map[string]any:
		res := make(map[string]any, len(v))
		for k, item := range v {
			interpItem, err := InterpolateValue(item, state)
			if err != nil {
				return nil, err
			}
			res[k] = interpItem
		}
		return res, nil
	case []any:
		res := make([]any, len(v))
		for i, item := range v {
			interpItem, err := InterpolateValue(item, state)
			if err != nil {
				return nil, err
			}
			res[i] = interpItem
		}
		return res, nil
	default:
		return v, nil
	}
}
