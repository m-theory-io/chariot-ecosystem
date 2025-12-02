package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
)

const (
	Prefix string = "CHARIOT_"
)

type Config struct {
	Headless       bool   `evar:"headless"`         // Headless mode for the application
	DevRESTEnabled bool   `evar:"dev_rest_enabled"` // Enable Dev REST API server (can run with or without headless)
	Port           int    `evar:"port"`             // Port for the application to listen on
	Timeout        int    `evar:"timeout"`          // Timeout for operations in seconds
	Verbose        bool   `evar:"verbose"`          // Verbose logging mode
	SSL            bool   `evar:"ssl"`              // Enable SSL/TLS (false for nginx termination)
	OnStart        string `evar:"on_start"`         // Command to execute on start
	OnExit         string `evar:"on_exit"`          // Command to execute on exit
	// Couchbase
	CBUrl      string `evar:"couchbase_url"`      // Couchbase connection URL
	CBUser     string `evar:"couchbase_user"`     // Couchbase username
	CBPassword string `evar:"couchbase_password"` // Couchbase password
	CBBucket   string `evar:"couchbase_bucket"`   // Couchbase bucket name
	CBScope    string `evar:"couchbase_scope"`    // Couchbase scope name
	CBDL       bool   `evar:"couchbase_cbdl"`     // Couchbase diagnostic log
	// RDBMS
	SQLDriver   string `evar:"sql_driver"`   // SQL driver type (e.g., mysql, postgres)
	SQLHost     string `evar:"sql_host"`     // SQL connection host
	SQLUser     string `evar:"sql_user"`     // SQL username
	SQLPassword string `evar:"sql_password"` // SQL password
	SQLDatabase string `evar:"sql_database"` // SQL database name
	SQLPort     int    `evar:"sql_port"`     // SQL port number
	// Vault
	VaultName      string `evar:"vault_name"`       // Azure Key Vault name
	VaultURI       string `evar:"vault_uri"`        // Azure Key Vault URI
	VaultKeyPrefix string `evar:"vault_key_prefix"` // Azure Key Vault key prefix (e.g., jpkey, docker)
	SecretProvider string `evar:"secret_provider"`  // Secret provider identifier (azure, file, etc.)
	SecretFilePath string `evar:"secret_file_path"` // Path to local secret file when using file provider
	// Serialization
	DataPath string `evar:"data_path"` // Path to store serialized data
	// Tree serialization
	TreePath   string `evar:"tree_path"`   // Path to store serialized tree data
	TreeFormat string `evar:"tree_format"` // Format for tree serialization (json, gob, etc.)
	// Diagram path
	DiagramPath string `evar:"diagram_path"` // Path to store VisualDSL diagrams
	// Cert path
	CertPath string `evar:"cert_path"` // Path to store certificates
	// Function library
	FunctionLib string `evar:"function_lib"` // Filename of the function library
	Bootstrap   string `evar:"bootstrap"`    // Bootstrap script to run on startup
	// Listeners registry persistence file (under data path)
	ListenersFile string `evar:"listeners_file"`
	// MCP (Model Context Protocol) integration
	MCPEnabled   bool   `evar:"mcp_enabled"`   // Enable MCP server
	MCPTransport string `evar:"mcp_transport"` // stdio | ws (websocket)
	MCPWSPath    string `evar:"mcp_ws_path"`   // WebSocket path when using ws
}

var ChariotConfig = &Config{}
var ChariotLogger *logs.ZapLogger
var ChariotKey = "BF0CB725-1AFE-4EB5-B06C-0AA0A778C2FA"

// StringVar reads an environment variable and sets it in the Config struct.
// If the environment variable is not set, it assigns a default value.
func (c *Config) StringVar(evar string, receiver *string, defValue string) {
	if evar == "" || receiver == nil {
		return // No operation if evar is empty or receiver is nil
	}
	if !strings.HasPrefix(evar, Prefix) {
		evar = Prefix + strings.ToUpper(evar)
	}
	var eValue interface{}
	eValue = os.Getenv(evar)
	if tval, ok := eValue.(string); ok && tval != "" {
		*receiver = tval
	} else {
		*receiver = defValue
	}
}

// IntVar reads an environment variable and sets it in the Config struct.
func (c *Config) IntVar(evar string, receiver *int, defValue int) {
	if evar == "" || receiver == nil {
		return // No operation if evar is empty or receiver is nil
	}
	if !strings.HasPrefix(evar, Prefix) {
		evar = Prefix + strings.ToUpper(evar)
	}
	var eValue interface{}
	eValue = os.Getenv(evar)
	if tval, ok := eValue.(string); ok && tval != "" {
		if val, err := strconv.Atoi(tval); err == nil {
			*receiver = val
		} else {
			*receiver = defValue
		}
	} else {
		*receiver = defValue
	}
}

// BoolVar reads an environment variable and sets it in the Config struct.
func (c *Config) BoolVar(evar string, receiver *bool, defValue bool) {
	if evar == "" || receiver == nil {
		return // No operation if evar is empty or receiver is nil
	}
	if !strings.HasPrefix(evar, Prefix) {
		evar = Prefix + strings.ToUpper(evar)
	}
	var eValue interface{}
	eValue = os.Getenv(evar)
	if tval, ok := eValue.(string); ok && tval != "" {
		if val, err := strconv.ParseBool(tval); err == nil {
			*receiver = val
		} else {
			*receiver = defValue
		}
	} else {
		*receiver = defValue
	}
}

// FloatVar reads an environment variable and sets it in the Config struct.
func (c *Config) FloatVar(evar string, receiver *float64, defValue float64) {
	if evar == "" || receiver == nil {
		return // No operation if evar is empty or receiver is nil
	}
	if !strings.HasPrefix(evar, Prefix) {
		evar = Prefix + strings.ToUpper(evar)
	}
	var eValue interface{}
	eValue = os.Getenv(evar)
	if tval, ok := eValue.(string); ok && tval != "" {
		if val, err := strconv.ParseFloat(tval, 64); err == nil {
			*receiver = val
		} else {
			*receiver = defValue
		}
	} else {
		*receiver = defValue
	}
}

// expandUserPath expands a leading ~ to the current user's home directory.
// If the path doesn't start with ~, it's returned as-is.
func expandUserPath(p string) string {
	if p == "" {
		return p
	}
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			// handle cases: "~", "~/..."
			if p == "~" {
				return home
			}
			// replace only the leading tilde
			return filepath.Join(home, strings.TrimPrefix(p, "~/"))
		}
	}
	return p
}

// ExpandAndNormalizePaths expands ~ and cleans configured filesystem paths.
// Call this after binding environment variables.
func ExpandAndNormalizePaths() {
	// Expand
	ChariotConfig.DataPath = expandUserPath(ChariotConfig.DataPath)
	ChariotConfig.TreePath = expandUserPath(ChariotConfig.TreePath)
	ChariotConfig.DiagramPath = expandUserPath(ChariotConfig.DiagramPath)
	ChariotConfig.CertPath = expandUserPath(ChariotConfig.CertPath)

	// Clean and, if relative, make absolute relative to current working directory
	normalize := func(p string) string {
		if p == "" {
			return p
		}
		p = filepath.Clean(p)
		if !filepath.IsAbs(p) {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
		}
		return p
	}

	ChariotConfig.DataPath = normalize(ChariotConfig.DataPath)
	ChariotConfig.TreePath = normalize(ChariotConfig.TreePath)
	ChariotConfig.DiagramPath = normalize(ChariotConfig.DiagramPath)
	ChariotConfig.CertPath = normalize(ChariotConfig.CertPath)
}
