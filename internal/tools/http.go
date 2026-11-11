package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ExecuteHTTPRequest performs an outbound HTTP REST or GraphQL call.
func ExecuteHTTPRequest(ctx context.Context, params map[string]any) (any, error) {
	urlVal, ok := params["url"].(string)
	if !ok || urlVal == "" {
		return nil, fmt.Errorf("http_request requires 'url' parameter")
	}

	methodVal, _ := params["method"].(string)
	if methodVal == "" {
		methodVal = "GET"
	}
	methodVal = strings.ToUpper(methodVal)

	timeout := 30 * time.Second
	if tVal, ok := params["timeout"].(string); ok {
		if parsed, err := time.ParseDuration(tVal); err == nil {
			timeout = parsed
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if bodyVal, exists := params["body"]; exists && bodyVal != nil {
		switch b := bodyVal.(type) {
		case string:
			bodyReader = strings.NewReader(b)
		default:
			bytesData, err := json.Marshal(b)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bytesData)
		}
	}

	req, err := http.NewRequestWithContext(reqCtx, methodVal, urlVal, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	// Apply headers
	if headersVal, exists := params["headers"]; exists && headersVal != nil {
		switch h := headersVal.(type) {
		case map[string]any:
			for k, v := range h {
				req.Header.Set(k, fmt.Sprintf("%v", v))
			}
		case map[string]string:
			for k, v := range h {
				req.Header.Set(k, v)
			}
		}
	}

	if req.Header.Get("Content-Type") == "" && bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var parsedBody any
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") || json.Valid(respBytes) {
		var j any
		if err := json.Unmarshal(respBytes, &j); err == nil {
			parsedBody = j
		} else {
			parsedBody = string(respBytes)
		}
	} else {
		parsedBody = string(respBytes)
	}

	result := map[string]any{
		"status_code": resp.StatusCode,
		"body":        parsedBody,
		"headers":     resp.Header,
	}

	if resp.StatusCode >= 400 {
		return result, fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, string(respBytes))
	}

	return result, nil
}
