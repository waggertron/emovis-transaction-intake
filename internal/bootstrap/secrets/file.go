package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const MaxFileBytes = 64 * 1024

type FileProvider struct {
	path string
}

func NewFileProvider(path string) *FileProvider {
	return &FileProvider{path: path}
}

func (provider *FileProvider) Load(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load local secret file: %w", err)
	}
	file, err := os.Open(provider.path)
	if err != nil {
		return nil, fmt.Errorf("open local secret file: %w", err)
	}
	defer file.Close()

	payload, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read local secret file: %w", err)
	}
	if len(payload) > MaxFileBytes {
		return nil, fmt.Errorf("local secret file exceeds %d bytes", MaxFileBytes)
	}
	values := map[string]string{}
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, fmt.Errorf("decode local secret file: invalid JSON")
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("local secret file contains no values")
	}
	for key, value := range values {
		if strings.TrimSpace(key) == "" || value == "" {
			return nil, fmt.Errorf("local secret file contains an empty key or value")
		}
	}
	return values, nil
}

func Overlay(base func(string) string, values map[string]string) func(string) string {
	return func(key string) string {
		if value := base(key); value != "" {
			return value
		}
		return values[key]
	}
}
