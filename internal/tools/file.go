package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ExecuteFileRead reads file content from disk.
func ExecuteFileRead(ctx context.Context, params map[string]any) (any, error) {
	pathVal, ok := params["path"].(string)
	if !ok || pathVal == "" {
		return nil, fmt.Errorf("file_read requires 'path' parameter")
	}

	cleanPath := filepath.Clean(pathVal)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", cleanPath, err)
	}

	return string(data), nil
}

// ExecuteFileWrite writes content to a file on disk.
func ExecuteFileWrite(ctx context.Context, params map[string]any) (any, error) {
	pathVal, ok := params["path"].(string)
	if !ok || pathVal == "" {
		return nil, fmt.Errorf("file_write requires 'path' parameter")
	}

	contentVal, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("file_write requires 'content' string parameter")
	}

	cleanPath := filepath.Clean(pathVal)
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(cleanPath, []byte(contentVal), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file %s: %w", cleanPath, err)
	}

	return map[string]any{
		"path":  cleanPath,
		"bytes": len(contentVal),
	}, nil
}
