package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/application"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/vault"
	"go.uber.org/zap"

	mcpserver "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/mcp"
	"github.com/bhouse1273/kissflag"
)

func init() {
	kissflag.SetPrefix("CHARIOT_")

	// Read configuration from environment variables
	cfg.ChariotConfig.BoolVar("headless", &cfg.ChariotConfig.Headless, false)
	// Dev REST can be toggled independently of headless; default true to preserve current behavior
	cfg.ChariotConfig.BoolVar("dev_rest_enabled", &cfg.ChariotConfig.DevRESTEnabled, true)
	cfg.ChariotConfig.IntVar("port", &cfg.ChariotConfig.Port, 8087)
	cfg.ChariotConfig.IntVar("timeout", &cfg.ChariotConfig.Timeout, 30)
	cfg.ChariotConfig.IntVar("shutdown_timeout", &cfg.ChariotConfig.ShutdownTimeout, 30)
	cfg.ChariotConfig.BoolVar("verbose", &cfg.ChariotConfig.Verbose, false)
	cfg.ChariotConfig.BoolVar("ssl", &cfg.ChariotConfig.SSL, false)
	// Couchbase connectivity
	cfg.ChariotConfig.StringVar("couchbase_url", &cfg.ChariotConfig.CBUrl, "localhost")
	cfg.ChariotConfig.StringVar("couchbase_user", &cfg.ChariotConfig.CBUser, "")
	cfg.ChariotConfig.StringVar("couchbase_password", &cfg.ChariotConfig.CBPassword, "")
	cfg.ChariotConfig.StringVar("couchbase_bucket", &cfg.ChariotConfig.CBBucket, "chariot")
	cfg.ChariotConfig.StringVar("couchbase_scope", &cfg.ChariotConfig.CBScope, "_default")
	cfg.ChariotConfig.BoolVar("couchbase_cbdl", &cfg.ChariotConfig.CBDL, false)
	// MySQL specific configuration
	cfg.ChariotConfig.StringVar("sql_driver", &cfg.ChariotConfig.SQLDriver, "mysql")
	cfg.ChariotConfig.StringVar("sql_host", &cfg.ChariotConfig.SQLHost, "")
	cfg.ChariotConfig.StringVar("sql_user", &cfg.ChariotConfig.SQLUser, "")
	cfg.ChariotConfig.StringVar("sql_password", &cfg.ChariotConfig.SQLPassword, "")
	cfg.ChariotConfig.StringVar("sql_database", &cfg.ChariotConfig.SQLDatabase, "")
	cfg.ChariotConfig.IntVar("sql_port", &cfg.ChariotConfig.SQLPort, 3306)
	// Vault configuration
	cfg.ChariotConfig.StringVar("vault_name", &cfg.ChariotConfig.VaultName, "chariot-vault")
	cfg.ChariotConfig.StringVar("vault_key_prefix", &cfg.ChariotConfig.VaultKeyPrefix, "jpkey")
	cfg.ChariotConfig.StringVar("secret_provider", &cfg.ChariotConfig.SecretProvider, "azure")
	cfg.ChariotConfig.StringVar("secret_file_path", &cfg.ChariotConfig.SecretFilePath, "./config/secrets.local.json")
	// File serialization path
	cfg.ChariotConfig.StringVar("data_path", &cfg.ChariotConfig.DataPath, "./data")
	// Tree serialization path
	cfg.ChariotConfig.StringVar("tree_path", &cfg.ChariotConfig.TreePath, "./data/trees")
	// Tree serialization format
	cfg.ChariotConfig.StringVar("tree_format", &cfg.ChariotConfig.TreeFormat, "gob")
	// Diagram serialization path
	cfg.ChariotConfig.StringVar("diagram_path", &cfg.ChariotConfig.DiagramPath, "./data/diagrams")
	// Cert path
	cfg.ChariotConfig.StringVar("cert_path", &cfg.ChariotConfig.CertPath, "../.certs")
	// Sandbox configuration
	cfg.ChariotConfig.BoolVar("sandbox_enabled", &cfg.ChariotConfig.SandboxEnabled, false)
	cfg.ChariotConfig.StringVar("sandbox_root", &cfg.ChariotConfig.SandboxRoot, "./data/sandboxes")
	cfg.ChariotConfig.StringVar("sandbox_default_scope", &cfg.ChariotConfig.SandboxDefaultScope, "sandbox")
	// Function library
	cfg.ChariotConfig.StringVar("function_lib", &cfg.ChariotConfig.FunctionLib, "stlib.json")
	// Bootstrap script
	cfg.ChariotConfig.StringVar("bootstrap", &cfg.ChariotConfig.Bootstrap, "bootstrap.ch")
	cfg.ChariotConfig.BoolVar("bootstrap_edit_enabled", &cfg.ChariotConfig.BootstrapEditEnabled, false)
	// Listeners registry file (under data path by default)
	cfg.ChariotConfig.StringVar("listeners_file", &cfg.ChariotConfig.ListenersFile, "listeners.json")
	// Directory for dedicated listener scripts
	cfg.ChariotConfig.StringVar("listener_scripts_dir", &cfg.ChariotConfig.ListenerScriptsDir, "listeners")
	// MCP configuration
	cfg.ChariotConfig.BoolVar("mcp_enabled", &cfg.ChariotConfig.MCPEnabled, false)
	cfg.ChariotConfig.BoolVar("mcp_execute_enabled", &cfg.ChariotConfig.MCPExecuteEnabled, false)
	cfg.ChariotConfig.StringVar("mcp_transport", &cfg.ChariotConfig.MCPTransport, "stdio")
	cfg.ChariotConfig.StringVar("mcp_ws_path", &cfg.ChariotConfig.MCPWSPath, "/mcp")
	// NSQ messaging
	cfg.ChariotConfig.BoolVar("nsq_enabled", &cfg.ChariotConfig.NSQEnabled, true)
	cfg.ChariotConfig.StringVar("nsq_addr", &cfg.ChariotConfig.NSQDAddress, "localhost:4150")
	cfg.ChariotConfig.StringVar("nsq_lookupd", &cfg.ChariotConfig.NSQLookupdAddress, "localhost:4160")
	cfg.ChariotConfig.StringVar("nsq_topic", &cfg.ChariotConfig.NSQDefaultTopic, "chariot_requests")
	cfg.ChariotConfig.StringVar("nsq_channel", &cfg.ChariotConfig.NSQDefaultChannel, "chariot_channel")
	cfg.ChariotConfig.StringVar("nsq_response_topics", &cfg.ChariotConfig.NSQResponseTopics, "default:chariot_responses")

	// Bind evars
	_ = kissflag.BindAllEVars(cfg.ChariotConfig)
	// Normalize any configured paths (expand ~, make absolute, clean)
	cfg.ExpandAndNormalizePaths()

	// Policy: do not accept legacy env var names. If legacy is present, warn and ignore.
	if legacy := os.Getenv("CHARIOT_BOOTSTRAP_FILE"); legacy != "" {
		// No functional fallback here by design – prefer fixing envs to match canonical name.
		// We'll log a warning at runtime in main() as well (once logger is ready).
	}

}

func main() {
	slogger := logs.NewZapLogger()
	cfg.ChariotLogger = slogger
	if err := run(slogger); err != nil {
		slogger.Error("Chariot service stopped with an error", zap.Error(err))
		_ = slogger.Sync()
		os.Exit(1)
	}
	_ = slogger.Sync()
}

func run(slogger *logs.ZapLogger) error {
	slogger.Info("Starting Chariot service beginning...")
	if legacy := os.Getenv("CHARIOT_BOOTSTRAP_FILE"); legacy != "" {
		slogger.Warn("Ignoring legacy env var CHARIOT_BOOTSTRAP_FILE; use CHARIOT_BOOTSTRAP instead",
			zap.String("legacy_value", legacy),
			zap.String("canonical_var", "CHARIOT_BOOTSTRAP"),
			zap.String("canonical_value", cfg.ChariotConfig.Bootstrap))
	}
	slogger.Info("Chariot configuration",
		zap.String("DataPath", cfg.ChariotConfig.DataPath),
		zap.String("TreePath", cfg.ChariotConfig.TreePath),
		zap.String("DiagramPath", cfg.ChariotConfig.DiagramPath),
		zap.String("FunctionLib", cfg.ChariotConfig.FunctionLib),
		zap.String("Bootstrap", cfg.ChariotConfig.Bootstrap),
		zap.Bool("BootstrapEditEnabled", cfg.ChariotConfig.BootstrapEditEnabled),
		zap.Bool("MCPExecuteEnabled", cfg.ChariotConfig.MCPExecuteEnabled),
		zap.Bool("Headless", cfg.ChariotConfig.Headless),
		zap.Bool("DevRESTEnabled", cfg.ChariotConfig.DevRESTEnabled),
		zap.Int("ShutdownTimeoutSeconds", cfg.ChariotConfig.ShutdownTimeout))

	if spec := strings.TrimSpace(cfg.ChariotConfig.NSQResponseTopics); spec != "" {
		if err := chariot.RegisterResponseTopicsFromSpec(spec); err != nil {
			return fmt.Errorf("register NSQ response topics: %w", err)
		}
		cfg.ChariotLogger.Info("Registered NSQ response topics", zap.String("spec", spec))
	}
	if err := vault.InitVaultClient(); err != nil {
		return fmt.Errorf("initialize Vault client: %w", err)
	}

	// Stdio remains an explicit tooling mode and does not launch application listeners.
	if cfg.ChariotConfig.MCPEnabled && strings.EqualFold(cfg.ChariotConfig.MCPTransport, "stdio") {
		cfg.ChariotLogger.Info("Starting MCP (stdio) server")
		if err := mcpserver.RunSTDIO(); err != nil {
			return fmt.Errorf("MCP stdio server: %w", err)
		}
		return nil
	}
	if !cfg.ChariotConfig.Headless && !cfg.ChariotConfig.DevRESTEnabled {
		cfg.ChariotLogger.Info("No application service surface is enabled; exiting")
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	applicationRuntime, err := application.BuildApplicationRuntime(ctx, application.RuntimeOptions{
		FunctionLibrary: cfg.ChariotConfig.FunctionLib,
		Bootstrap:       cfg.ChariotConfig.Bootstrap,
	})
	if err != nil {
		// An incomplete rollback returns its still-owned runtime alongside the
		// startup error. Retry with fresh grace, not the cancelled startup context;
		// arbitrary script execution must not hold process failure open forever.
		return errors.Join(fmt.Errorf("initialize application runtime: %w", err), cleanupApplicationRuntime(applicationRuntime))
	}

	sessionManager := chariot.NewSessionManager(
		time.Duration(cfg.ChariotConfig.Timeout)*time.Minute,
		5*time.Minute)
	host, err := application.NewHost(application.HostOptions{
		Runtime:         applicationRuntime,
		SessionManager:  sessionManager,
		Logger:          slogger,
		DevRESTEnabled:  cfg.ChariotConfig.DevRESTEnabled,
		Port:            cfg.ChariotConfig.Port,
		SSL:             cfg.ChariotConfig.SSL,
		CertPath:        cfg.ChariotConfig.CertPath,
		MCPEnabled:      cfg.ChariotConfig.MCPEnabled,
		MCPTransport:    cfg.ChariotConfig.MCPTransport,
		MCPPath:         cfg.ChariotConfig.MCPWSPath,
		ShutdownTimeout: time.Duration(cfg.ChariotConfig.ShutdownTimeout) * time.Second,
	})
	if err != nil {
		cleanupErr := cleanupApplicationRuntime(applicationRuntime)
		sessionManager.Stop()
		return errors.Join(fmt.Errorf("create application host: %w", err), cleanupErr)
	}
	return host.Run(ctx)
}

// Keep the runtime referenced throughout a bounded final cleanup attempt. A
// timeout does not prove blocked scripts have exited; main reports failure and
// exits rather than waiting indefinitely for those scripts.
func cleanupApplicationRuntime(applicationRuntime *application.ApplicationRuntime) error {
	if applicationRuntime == nil || applicationRuntime.Runtime == nil {
		return nil
	}
	grace := time.Duration(cfg.ChariotConfig.ShutdownTimeout) * time.Second
	if grace <= 0 {
		grace = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := applicationRuntime.Runtime.ShutdownListeners(ctx); err != nil {
		return fmt.Errorf("cleanup application runtime: %w", err)
	}
	return nil
}
