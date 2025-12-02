package vault

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func TestFileProviderLoadsSecrets(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	secretPath := filepath.Join(tempDir, "secrets.json")
	payload := []byte(`{"alpha": {"answer": 42}, "beta": "plain"}`)
	if err := os.WriteFile(secretPath, payload, 0o600); err != nil {
		t.Fatalf("failed to write temp secret file: %v", err)
	}

	cfg.ChariotConfig.SecretFilePath = secretPath

	provider := newFileProvider()
	if err := provider.Init(context.Background()); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	got, err := provider.GetSecret(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("alpha lookup failed: %v", err)
	}
	if got != `{"answer":42}` {
		t.Fatalf("unexpected alpha payload: %s", got)
	}

	plain, err := provider.GetSecret(context.Background(), "beta")
	if err != nil {
		t.Fatalf("beta lookup failed: %v", err)
	}
	if plain != "plain" {
		t.Fatalf("unexpected beta payload: %s", plain)
	}
}
