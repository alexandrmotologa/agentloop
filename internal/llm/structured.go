package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	codeFenceRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
)

// ExtractJSON cleans conversational wrapper text and extracts valid JSON payload.
func ExtractJSON(content string) (string, error) {
	trimmed := strings.TrimSpace(content)

	// Check for markdown code fences
	matches := codeFenceRegex.FindStringSubmatch(trimmed)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1]), nil
	}

	// Find the first '{' and last '}'
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start != -1 && end != -1 && end > start {
		return strings.TrimSpace(trimmed[start : end+1]), nil
	}

	// Check if array JSON
	arrStart := strings.Index(trimmed, "[")
	arrEnd := strings.LastIndex(trimmed, "]")
	if arrStart != -1 && arrEnd != -1 && arrEnd > arrStart {
		return strings.TrimSpace(trimmed[arrStart : arrEnd+1]), nil
	}

	return trimmed, nil
}

// ValidateJSONAgainstSchema validates a JSON object against a JSON Schema map.
func ValidateJSONAgainstSchema(data map[string]any, schemaMap map[string]any) ([]string, error) {
	if schemaMap == nil {
		return nil, nil
	}

	schemaBytes, err := json.Marshal(schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	if err := compiler.AddResource("schema.json", strings.NewReader(string(schemaBytes))); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data for validation: %w", err)
	}

	var rawData any
	if err := json.Unmarshal(dataBytes, &rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data for validation: %w", err)
	}

	if err := schema.Validate(rawData); err != nil {
		var validationErrors []string
		if ve, ok := err.(*jsonschema.ValidationError); ok {
			for _, subErr := range ve.Causes {
				validationErrors = append(validationErrors, subErr.Error())
			}
			if len(validationErrors) == 0 {
				validationErrors = append(validationErrors, ve.Error())
			}
		} else {
			validationErrors = append(validationErrors, err.Error())
		}
		return validationErrors, nil
	}

	return nil, nil
}

// ParseAndValidate extracts JSON from content, unmarshals it, and checks schema conformance.
func ParseAndValidate(content string, schema map[string]any) (map[string]any, []string, error) {
	cleaned, err := ExtractJSON(content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract json: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(cleaned), &data); err != nil {
		return nil, []string{fmt.Sprintf("json syntax error: %v", err)}, fmt.Errorf("invalid json format: %w", err)
	}

	if schema == nil {
		return data, nil, nil
	}

	errs, err := ValidateJSONAgainstSchema(data, schema)
	if err != nil {
		return data, nil, err
	}

	return data, errs, nil
}
