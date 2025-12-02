package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"go.uber.org/zap"
)

type fileProvider struct {
	path    string
	secrets map[string]string
	mu      sync.RWMutex
}

func newFileProvider() SecretProvider {
	return &fileProvider{
		path:    cfg.ChariotConfig.SecretFilePath,
		secrets: make(map[string]string),
	}
}

func (p *fileProvider) Init(ctx context.Context) error {
	if p.path == "" {
		return fmt.Errorf("secret_file_path must be configured for file provider")
	}

	abs, err := filepath.Abs(p.path)
	if err != nil {
		return fmt.Errorf("resolve secret file path: %w", err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("read secret file '%s': %w", abs, err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse secret file '%s': %w", abs, err)
	}

	parsed := make(map[string]string, len(raw))
	for key, value := range raw {
		switch v := value.(type) {
		case string:
			parsed[key] = v
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				return fmt.Errorf("encode secret '%s': %w", key, err)
			}
			parsed[key] = string(bytes)
		}
	}

	p.mu.Lock()
	p.secrets = parsed
	p.path = abs
	p.mu.Unlock()

	if cfg.ChariotLogger != nil {
		cfg.ChariotLogger.Info("File secret provider loaded", zap.String("path", abs), zap.Int("secret_count", len(parsed)))
	}
	return nil
}

func (p *fileProvider) Name() string {
	return "file"
}

func (p *fileProvider) GetSecret(ctx context.Context, name string) (string, error) {
	p.mu.RLock()
	value, ok := p.secrets[name]
	p.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("secret '%s' not found in %s", name, p.path)
	}

	return value, nil
}
