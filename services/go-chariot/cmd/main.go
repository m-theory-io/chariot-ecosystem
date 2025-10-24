package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/listeners"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/vault"
	"go.uber.org/zap"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/handlers"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/routes"
	mcpserver "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/mcp"
	"github.com/bhouse1273/kissflag"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func init() {
	kissflag.SetPrefix("CHARIOT_")

	// Read configuration from environment variables
	cfg.ChariotConfig.BoolVar("headless", &cfg.ChariotConfig.Headless, false)
	// Dev REST can be toggled independently of headless; default true to preserve current behavior
	cfg.ChariotConfig.BoolVar("dev_rest_enabled", &cfg.ChariotConfig.DevRESTEnabled, true)
	cfg.ChariotConfig.IntVar("port", &cfg.ChariotConfig.Port, 8087)
	cfg.ChariotConfig.IntVar("timeout", &cfg.ChariotConfig.Timeout, 30)
	cfg.ChariotConfig.BoolVar("verbose", &cfg.ChariotConfig.Verbose, false)
	cfg.ChariotConfig.BoolVar("ssl", &cfg.ChariotConfig.SSL, false)
	// Couchbase connectivity
	cfg.ChariotConfig.StringVar("couchbase_url", &cfg.ChariotConfig.CBUrl, "192.168.0.117")
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
	// Function library
	cfg.ChariotConfig.StringVar("function_lib", &cfg.ChariotConfig.FunctionLib, "stlib.json")
	// Bootstrap script
	cfg.ChariotConfig.StringVar("bootstrap", &cfg.ChariotConfig.Bootstrap, "bootstrap.ch")
	// Listeners registry file (under data path by default)
	cfg.ChariotConfig.StringVar("listeners_file", &cfg.ChariotConfig.ListenersFile, "listeners.json")
	// MCP configuration
	cfg.ChariotConfig.BoolVar("mcp_enabled", &cfg.ChariotConfig.MCPEnabled, false)
	cfg.ChariotConfig.StringVar("mcp_transport", &cfg.ChariotConfig.MCPTransport, "ws")
	cfg.ChariotConfig.StringVar("mcp_ws_path", &cfg.ChariotConfig.MCPWSPath, "/mcp")

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
	defer slogger.Sync() // Ensure logger is flushed before exit
	cfg.ChariotLogger = slogger
	// Warn if legacy env var is present
	if legacy := os.Getenv("CHARIOT_BOOTSTRAP_FILE"); legacy != "" {
		slogger.Warn("Ignoring legacy env var CHARIOT_BOOTSTRAP_FILE; use CHARIOT_BOOTSTRAP instead",
			zap.String("legacy_value", legacy),
			zap.String("canonical_var", "CHARIOT_BOOTSTRAP"),
			zap.String("canonical_value", cfg.ChariotConfig.Bootstrap),
		)
	}
	// Log key configuration for diagnostics
	slogger.Info("Chariot configuration",
		zap.String("DataPath", cfg.ChariotConfig.DataPath),
		zap.String("TreePath", cfg.ChariotConfig.TreePath),
		zap.String("DiagramPath", cfg.ChariotConfig.DiagramPath),
		zap.String("FunctionLib", cfg.ChariotConfig.FunctionLib),
		zap.String("Bootstrap", cfg.ChariotConfig.Bootstrap),
		zap.Bool("Headless", cfg.ChariotConfig.Headless),
		zap.Bool("DevRESTEnabled", cfg.ChariotConfig.DevRESTEnabled),
	)
	// Create session manager with 30 minute timeout, clean up every 5 minutes
	timeOut := time.Duration(cfg.ChariotConfig.Timeout) * time.Minute
	cleanUpInterval := time.Duration(5) * time.Minute
	sessionManager := chariot.NewSessionManager(timeOut, cleanUpInterval)
	if err := vault.InitVaultClient(); err != nil { // Initialize Azure Key Vault client
		cfg.ChariotLogger.Error("Failed to initialize Vault client", zap.Error(err))
		return
	}

	// Start MCP server in stdio mode if enabled, then exit (intended to be launched as a subprocess by clients)
	if cfg.ChariotConfig.MCPEnabled && strings.ToLower(cfg.ChariotConfig.MCPTransport) == "stdio" {
		cfg.ChariotLogger.Info("Starting MCP (stdio) server")
		if err := mcpserver.RunSTDIO(); err != nil {
			cfg.ChariotLogger.Error("MCP stdio server error", zap.Error(err))
		}
		return
	}

	// Optionally start headless session (does not block if Dev REST is also enabled)
	if cfg.ChariotConfig.Headless {
		// Initialize a bootstrap runtime that mirrors the REST handlers runtime
		bootstrapRuntime := chariot.NewRuntime()
		chariot.RegisterAll(bootstrapRuntime)

		// Load stdlib functions from configured library and register them
		if cfg.ChariotConfig.FunctionLib != "" {
			if funcs, err := chariot.LoadFunctionsFromFile(cfg.ChariotConfig.FunctionLib); err == nil {
				for name, fn := range funcs {
					bootstrapRuntime.RegisterFunction(name, fn)
				}
			} else {
				cfg.ChariotLogger.Warn("Failed to load function library", zap.String("file", cfg.ChariotConfig.FunctionLib), zap.Error(err))
			}
		}

		// Optionally load bootstrap script (users, helpers, etc.)
		if cfg.ChariotConfig.Bootstrap != "" {
			if fullPath, err := chariot.GetSecureFilePath(cfg.ChariotConfig.Bootstrap, "data"); err == nil {
				if content, err := os.ReadFile(fullPath); err == nil {
					if _, err := bootstrapRuntime.ExecProgram(string(content)); err != nil {
						cfg.ChariotLogger.Warn("Failed to execute bootstrap script in headless mode", zap.Error(err))
					}
				} else {
					cfg.ChariotLogger.Warn("Failed to read bootstrap script in headless mode", zap.Error(err))
				}
			} else {
				cfg.ChariotLogger.Warn("Failed to resolve bootstrap path in headless mode", zap.Error(err))
			}
		}

		// Initialize listeners manager and auto-start listeners marked AutoStart
		lman := listeners.NewManager(bootstrapRuntime)
		if err := lman.Load(); err != nil {
			cfg.ChariotLogger.Warn("Failed to load listeners registry in headless mode", zap.Error(err))
		}
		for _, l := range lman.List() {
			if l.AutoStart {
				if _, err := lman.Start(l.Name, cfg.ChariotConfig.Port); err != nil {
					cfg.ChariotLogger.Warn("Failed to auto-start listener (headless)", zap.String("name", l.Name), zap.Error(err))
				} else {
					cfg.ChariotLogger.Info("Auto-started listener (headless)", zap.String("name", l.Name))
				}
			}
		}
	}

	// Optionally start Dev REST API server
	if cfg.ChariotConfig.DevRESTEnabled {
		h := handlers.NewHandlers(sessionManager)
		e := echo.New()
		routes.RegisterRoutes(e, h)
		e.Use(middleware.Logger())
		e.Use(middleware.Recover())
		e.Use(logs.ZapLoggerMiddleware(slogger.Get()))

		// Add this middleware in your Echo setup
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				log.Printf("ECHO DEBUG: Received Authorization header: '%s'", c.Request().Header.Get("Authorization"))
				log.Printf("ECHO DEBUG: All headers: %+v", c.Request().Header)
				log.Printf("ECHO DEBUG: Method: %s, Path: %s", c.Request().Method, c.Request().URL.Path)
				return next(c)
			}
		})

		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
			AllowHeaders: []string{"Content-Type", "Authorization"}, // Make sure Authorization is here
		}))

		basePath := cfg.ChariotConfig.CertPath
		if basePath == "" {
			basePath = "./.certs" // Default path if not set
		}
		fullPathCrt := fmt.Sprintf("%s/server.crt", basePath)
		fullPathKey := fmt.Sprintf("%s/server.key", basePath)

		// debug
		fmt.Println("Current working directory:", func() string { dir, _ := os.Getwd(); return dir }())
		fmt.Println("Looking for cert at:", fullPathCrt)
		fmt.Println("Looking for key at:", fullPathKey)

		// Optionally mount MCP WebSocket endpoint (placeholder until implemented)
		if cfg.ChariotConfig.MCPEnabled && strings.ToLower(cfg.ChariotConfig.MCPTransport) == "ws" {
			e.GET(cfg.ChariotConfig.MCPWSPath, func(c echo.Context) error { return mcpserver.HandleWS(c) })
			cfg.ChariotLogger.Info("MCP WebSocket route enabled", zap.String("path", cfg.ChariotConfig.MCPWSPath))
		}

		// Start server with or without SSL (this call blocks)
		if cfg.ChariotConfig.SSL {
			fmt.Printf("Starting TLS server on port %d\n", cfg.ChariotConfig.Port)
			e.Logger.Fatal(e.StartTLS(fmt.Sprintf(":%d", cfg.ChariotConfig.Port), fullPathCrt, fullPathKey))
		} else {
			fmt.Printf("Starting HTTP server on port %d (SSL disabled for nginx termination)\n", cfg.ChariotConfig.Port)
			e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.ChariotConfig.Port)))
		}
	} else {
		// If REST is disabled and headless is enabled, keep the process alive
		if cfg.ChariotConfig.Headless {
			select {}
		}
	}
}
