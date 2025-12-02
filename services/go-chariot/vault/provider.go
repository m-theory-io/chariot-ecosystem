package vault

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"go.uber.org/zap"
)

// SecretProvider defines the minimal contract for loading secrets from any backend.
type SecretProvider interface {
	Init(ctx context.Context) error
	Name() string
	GetSecret(ctx context.Context, name string) (string, error)
}

var (
	providerMu     sync.RWMutex
	activeProvider SecretProvider
)

// InitVaultClient initializes the configured secret provider. Kept for backward compatibility.
func InitVaultClient() error {
	return InitSecretProvider(context.Background())
}

// InitSecretProvider allows callers to pass a custom context when initializing the provider.
func InitSecretProvider(ctx context.Context) error {
	providerMu.Lock()
	defer providerMu.Unlock()

	prov, err := buildProvider()
	if err != nil {
		return err
	}

	if err := prov.Init(ctx); err != nil {
		return err
	}

	activeProvider = prov
	if cfg.ChariotLogger != nil {
		cfg.ChariotLogger.Info("Secret provider initialized", zap.String("provider", prov.Name()))
	}

	return nil
}

// buildProvider selects the appropriate provider implementation based on configuration.
func buildProvider() (SecretProvider, error) {
	providerType := strings.TrimSpace(strings.ToLower(cfg.ChariotConfig.SecretProvider))
	if providerType == "" || providerType == "azure" {
		return newAzureProvider(), nil
	}
	if providerType == "file" || providerType == "filesystem" || providerType == "fs" {
		return newFileProvider(), nil
	}
	return nil, fmt.Errorf("unsupported secret provider '%s'", providerType)
}

func getProvider() (SecretProvider, error) {
	providerMu.RLock()
	defer providerMu.RUnlock()
	if activeProvider == nil {
		return nil, errors.New("secret provider not initialized")
	}
	return activeProvider, nil
}

// HasProvider reports whether a provider has been successfully initialized.
func HasProvider() bool {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return activeProvider != nil
}

// ProviderName returns the name of the currently active provider, or an empty string if none.
func ProviderName() string {
	providerMu.RLock()
	defer providerMu.RUnlock()
	if activeProvider == nil {
		return ""
	}
	return activeProvider.Name()
}
