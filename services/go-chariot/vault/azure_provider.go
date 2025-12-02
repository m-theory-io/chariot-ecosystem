package vault

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	azv "github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"go.uber.org/zap"
)

// azureProvider implements SecretProvider using Azure Key Vault.
type azureProvider struct {
	client   *azv.Client
	vaultURI string
}

func newAzureProvider() SecretProvider {
	return &azureProvider{}
}

func (p *azureProvider) Init(ctx context.Context) error {
	if cfg.ChariotConfig.VaultName == "" {
		cfg.ChariotConfig.VaultName = "chariot-vault"
	}

	// Allow overriding the full URI (useful for sovereign clouds or testing)
	vaultURI := cfg.ChariotConfig.VaultURI
	if vaultURI == "" {
		vaultURI = fmt.Sprintf("https://%s.vault.azure.net", cfg.ChariotConfig.VaultName)
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		if cfg.ChariotLogger != nil {
			cfg.ChariotLogger.Error("Failed to create Azure credential", zap.String("details", err.Error()))
		}
		return fmt.Errorf("azure credential: %w", err)
	}

	client, err := azv.NewClient(vaultURI, cred, nil)
	if err != nil {
		if cfg.ChariotLogger != nil {
			cfg.ChariotLogger.Error("Failed to create Key Vault client", zap.String("vault_uri", vaultURI), zap.String("details", err.Error()))
		}
		return fmt.Errorf("azure key vault client: %w", err)
	}

	p.client = client
	p.vaultURI = vaultURI

	if cfg.ChariotLogger != nil {
		cfg.ChariotLogger.Info("Azure Key Vault client ready", zap.String("vault_uri", vaultURI))
	}
	return nil
}

func (p *azureProvider) Name() string {
	return "azure"
}

func (p *azureProvider) GetSecret(ctx context.Context, name string) (string, error) {
	if p.client == nil {
		return "", fmt.Errorf("azure provider not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := p.client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", fmt.Errorf("azure get secret '%s': %w", name, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("azure secret '%s' has no value", name)
	}

	return *resp.Value, nil
}
