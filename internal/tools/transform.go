package tools

import (
	"context"
	"fmt"

	"github.com/expr-lang/expr"
)

// ExecuteJSONTransform evaluates an expression over input data to reshape or filter it.
func ExecuteJSONTransform(ctx context.Context, params map[string]any) (any, error) {
	exprStr, ok := params["expr"].(string)
	if !ok || exprStr == "" {
		return nil, fmt.Errorf("json_transform requires 'expr' parameter")
	}

	inputVal, exists := params["input"]
	if !exists {
		inputVal = params
	}

	env := map[string]any{
		"input": inputVal,
		"data":  inputVal,
	}
	if stateVal, exists := params["state"]; exists {
		env["state"] = stateVal
	}

	program, err := expr.Compile(exprStr, expr.Env(env))
	if err != nil {
		return nil, fmt.Errorf("failed to compile transform expression '%s': %w", exprStr, err)
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("failed to run transform expression: %w", err)
	}

	return output, nil
}
