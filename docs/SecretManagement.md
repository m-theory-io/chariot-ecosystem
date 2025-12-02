# Secret Management Providers

Chariot now supports pluggable secret providers so you can run locally without Azure Key Vault while keeping production deployments secure.

## Configuration

| Setting | Env Var | Default | Description |
|---------|---------|---------|-------------|
| `SecretProvider` | `CHARIOT_SECRET_PROVIDER` | `azure` | Provider identifier. Supported values: `azure`, `file`. |
| `SecretFilePath` | `CHARIOT_SECRET_FILE_PATH` | `./config/secrets.local.json` | Path to a JSON file used when `SecretProvider=file`. |
| `VaultName` | `CHARIOT_VAULT_NAME` | `chariot-vault` | Azure Key Vault name (still used by the Azure provider). |
| `VaultKeyPrefix` | `CHARIOT_VAULT_KEY_PREFIX` | `jpkey` | Prefix used when generating secret names such as `<prefix>-<org-key>`. |

Set these values via the existing env var mechanism (`CHARIOT_*`). Example Azure configuration:

```bash
export CHARIOT_SECRET_PROVIDER=azure
export CHARIOT_VAULT_NAME=my-prod-vault
export CHARIOT_VAULT_KEY_PREFIX=prodkey
```

## File Provider

The file provider lets you develop or run tests without Azure access. Create a JSON file that maps secret names to either raw strings or structured objects (which are marshaled back to JSON before being returned):

```json
{
  "local-BF0CB725-1AFE-4EB5-B06C-0AA0A778C2FA": {
    "org_key": "BF0CB725-1AFE-4EB5-B06C-0AA0A778C2FA",
    "cb_scope": "_default",
    "cb_user": "mtheory",
    "cb_password": "localpass",
    "cb_url": "couchbase://localhost",
    "cb_bucket": "chariot",
    "sql_host": "localhost",
    "sql_database": "chariot",
    "sql_user": "root",
    "sql_password": "rootpass",
    "sql_driver": "mysql",
    "sql_port": 3306
  },
  "dev-aes-key": "bXktc2VjcmV0LWFpcy1rZXk="
}
```

Then point Chariot at the file:

```bash
export CHARIOT_SECRET_PROVIDER=file
export CHARIOT_SECRET_FILE_PATH=./config/secrets.local.json
```

## Migration Notes

1. **Keep existing env vars** – `VaultName` and `VaultKeyPrefix` are still honored for backwards compatibility with Azure.
2. **Custom providers** – add a new `SecretProvider` implementation inside `services/go-chariot/vault` and reference it inside `buildProvider()`.
3. **CI/CD** – set `CHARIOT_SECRET_PROVIDER=file` with a generated secret file when running tests without Azure credentials.
