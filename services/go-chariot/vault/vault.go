package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"go.uber.org/zap"

	cfg "github.com/bhouse1273/go-chariot/configs"
	// Import go-chariot/vault package for vault client initialization
)

// DBContext represents database connection information for a tenant
type DBContext struct {
	OrgKey      string `json:"org_key"`
	CBUser      string `json:"cb_user"`
	CBPassword  string `json:"cb_password"`
	CBURL       string `json:"cb_url"`
	CBBucket    string `json:"cb_bucket"`
	CBScope     string `json:"cb_scope"`
	SQLUser     string `json:"sql_user"`
	SQLPassword string `json:"sql_password"`
	SQLHost     string `json:"sql_host"`
	SQLPort     int    `json:"sql_port"`
	SQLDatabase string `json:"sql_database"`
	SQLDriver   string `json:"sql_driver"`
}

// OrgSecret represents the secret structure stored in Azure Key Vault
type OrgSecret struct {
	OrgKey      string `json:"org_key"`
	CBScope     string `json:"cb_scope"`
	CBUser      string `json:"cb_user"`
	CBPassword  string `json:"cb_password"`
	CBURL       string `json:"cb_url"`
	CBBucket    string `json:"cb_bucket"`
	SQLHost     string `json:"sql_host"`
	SQLDatabase string `json:"sql_database"`
	SQLUser     string `json:"sql_user"`
	SQLPassword string `json:"sql_password"`
	SQLDriver   string `json:"sql_driver"`
	SQLPort     int    `json:"sql_port"` // Optional, can be 0 if not used
}

// Global vault client instance
var (
	VaultClient *azsecrets.Client
	VaultURI    string
)

// InitVaultClient initializes the Azure Key Vault client using DefaultAzureCredential
func InitVaultClient() error {
	const logName = "InitVaultClient"

	if cfg.ChariotConfig.VaultName == "" {
		return fmt.Errorf("%s - vault name not configured", logName)
	}

	// Use DefaultAzureCredential - works for both local dev and Azure VMs
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		cfg.ChariotLogger.Error("Failed to create Azure credential", zap.String("details", err.Error()))
		return fmt.Errorf("%s - failed to create Azure credential: %w", logName, err)
	}

	// Build vault URI from config
	vaultUri := fmt.Sprintf("https://%s.vault.azure.net", cfg.ChariotConfig.VaultName)

	// Create Key Vault client
	client, err := azsecrets.NewClient(vaultUri, cred, nil)
	if err != nil {
		cfg.ChariotLogger.Error("Failed to create Key Vault client", zap.String("vault_uri", vaultUri), zap.String("details", err.Error()))
		return fmt.Errorf("%s - failed to create Key Vault client: %w", logName, err)
	}

	// Initialize global variables
	VaultClient = client
	VaultURI = vaultUri

	cfg.ChariotLogger.Info("Successfully initialized Key Vault client", zap.String("vault_uri", vaultUri))
	return nil
}

// GetOrgSecret retrieves and parses organization secret from Azure Key Vault
func GetOrgSecret(ctx context.Context, orgKey string) (*OrgSecret, error) {
	const logName = "GetOrgSecret"

	if VaultClient == nil {
		return nil, fmt.Errorf("%s - vault client not initialized", logName)
	}

	if orgKey == "" {
		return nil, fmt.Errorf("%s - orgKey is required", logName)
	}

	// Use the provided context directly (bootstrap context)
	secretName := makeSecretName(orgKey)
	resp, err := VaultClient.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		cfg.ChariotLogger.Error("Failed to get secret from vault", zap.String("org_key", orgKey), zap.String("secret_name", secretName), zap.String("details", err.Error()))
		return nil, fmt.Errorf("%s - failed to get secret for org %s: %w", logName, orgKey, err)
	}

	if resp.Value == nil {
		cfg.ChariotLogger.Error("Secret value is nil", zap.String("org_key", orgKey), zap.String("secret_name", secretName))
		return nil, fmt.Errorf("%s - secret value is nil for org: %s", logName, orgKey)
	}

	// Parse JSON secret value
	var orgSecret OrgSecret
	err = json.Unmarshal([]byte(*resp.Value), &orgSecret)
	if err != nil {
		cfg.ChariotLogger.Error("Failed to parse secret JSON", zap.String("org_key", orgKey), zap.String("details", err.Error()))
		return nil, fmt.Errorf("%s - failed to parse secret for org %s: %w", logName, orgKey, err)
	}

	cfg.ChariotLogger.Info("Parsed vault secret",
		zap.String("org_key", orgKey),
		zap.String("sql_host", orgSecret.SQLHost),
		zap.Int("sql_port", orgSecret.SQLPort),
		zap.String("sql_driver", orgSecret.SQLDriver))

	if cfg.ChariotConfig.Verbose {
		cfg.ChariotLogger.Info("Successfully retrieved secret", zap.String("org_key", orgKey))
	}

	return &orgSecret, nil
}

// GetSecretValue retrieves a simple string secret from Azure Key Vault
func GetSecretValue(ctx context.Context, secretName string) (string, error) {
	const logName = "GetSecretValue"

	if VaultClient == nil {
		return "", fmt.Errorf("%s - vault client not initialized", logName)
	}

	if secretName == "" {
		return "", fmt.Errorf("%s - secretName is required", logName)
	}

	// Retrieve secret from Key Vault (latest version)
	resp, err := VaultClient.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		cfg.ChariotLogger.Error("Failed to get secret from vault", zap.String("secret_name", secretName), zap.String("details", err.Error()))
		return "", fmt.Errorf("%s - failed to get secret %s: %w", logName, secretName, err)
	}

	if resp.Value == nil {
		cfg.ChariotLogger.Error("Secret value is nil", zap.String("secret_name", secretName))
		return "", fmt.Errorf("%s - secret value is nil for: %s", logName, secretName)
	}

	if cfg.ChariotConfig.Verbose {
		cfg.ChariotLogger.Info("Successfully retrieved secret", zap.String("secret_name", secretName))
	}

	return *resp.Value, nil
}

// ConvertOrgSecretToDBContext converts an OrgSecret to DBContext format
func ConvertOrgSecretToDBContext(orgKey string, orgSecret *OrgSecret) *DBContext {
	if orgSecret == nil {
		return nil
	}

	return &DBContext{
		OrgKey:      orgKey,
		CBUser:      orgSecret.CBUser,
		CBPassword:  orgSecret.CBPassword,
		CBURL:       orgSecret.CBURL,
		CBBucket:    orgSecret.CBBucket,
		CBScope:     orgSecret.CBScope,
		SQLUser:     orgSecret.SQLUser,
		SQLPassword: orgSecret.SQLPassword,
		SQLHost:     orgSecret.SQLHost,
		SQLDatabase: orgSecret.SQLDatabase,
		SQLDriver:   orgSecret.SQLDriver,
		SQLPort:     orgSecret.SQLPort,
	}
}

// ValidateOrgSecret validates that required fields are present in an OrgSecret
func ValidateOrgSecret(orgSecret *OrgSecret) error {
	if orgSecret == nil {
		return fmt.Errorf("orgSecret is nil")
	}

	// Check required Couchbase fields
	if orgSecret.CBURL == "" {
		return fmt.Errorf("CBURL is required")
	}
	if orgSecret.CBBucket == "" {
		return fmt.Errorf("CBBucket is required")
	}
	if orgSecret.CBScope == "" {
		return fmt.Errorf("CBScope is required")
	}
	if orgSecret.CBUser == "" {
		return fmt.Errorf("CBUser is required")
	}
	if orgSecret.CBPassword == "" {
		return fmt.Errorf("CBPassword is required")
	}

	// Check required SQL fields (if SQL is configured)
	if orgSecret.SQLHost != "" {
		if orgSecret.SQLDatabase == "" {
			return fmt.Errorf("SQLDatabase is required when SQLHost is specified")
		}
		if orgSecret.SQLUser == "" {
			return fmt.Errorf("SQLUser is required when SQLHost is specified")
		}
		if orgSecret.SQLPassword == "" {
			return fmt.Errorf("SQLPassword is required when SQLHost is specified")
		}
		if orgSecret.SQLDriver == "" {
			return fmt.Errorf("SQLDriver is required when SQLHost is specified")
		}
		if orgSecret.SQLPort <= 0 {
			return fmt.Errorf("SQLPort must be a positive integer when SQLHost is specified")
		}
	}

	return nil
}

func makeSecretName(orgKey string) string {
	// Normalize orgKey to lowercase and replace spaces with underscores
	// orgKey has the form <uuid>
	keyPrefix := cfg.ChariotConfig.VaultKeyPrefix
	if keyPrefix == "" {
		keyPrefix = "jpkey" // Default fallback
	}
	normalizedKey := fmt.Sprintf("%s-%s", keyPrefix, orgKey)
	return normalizedKey
}
