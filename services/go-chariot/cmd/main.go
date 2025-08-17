package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bhouse1273/go-chariot/chariot"
	cfg "github.com/bhouse1273/go-chariot/configs"
	"github.com/bhouse1273/go-chariot/logs"
	"github.com/bhouse1273/go-chariot/vault"
	"go.uber.org/zap"

	"github.com/bhouse1273/go-chariot/internal/handlers"
	"github.com/bhouse1273/go-chariot/internal/routes"
	"github.com/bhouse1273/kissflag"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func init() {
	kissflag.SetPrefix("CHARIOT_")

	// Read configuration from environment variables
	cfg.ChariotConfig.BoolVar("headless", &cfg.ChariotConfig.Headless, false)
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
	// Cert path
	cfg.ChariotConfig.StringVar("cert_path", &cfg.ChariotConfig.CertPath, "../.certs")
	// Function library
	cfg.ChariotConfig.StringVar("function_lib", &cfg.ChariotConfig.FunctionLib, "stlib.json")
	// Bootstrap script
	cfg.ChariotConfig.StringVar("bootstrap", &cfg.ChariotConfig.Bootstrap, "bootstrap.ch")

	// Bind evars
	kissflag.BindAllEVars(cfg.ChariotConfig)

}

func main() {
	slogger := logs.NewZapLogger()
	defer slogger.Sync() // Ensure logger is flushed before exit
	cfg.ChariotLogger = slogger
	// Create session manager with 30 minute timeout, clean up every 5 minutes
	timeOut := time.Duration(cfg.ChariotConfig.Timeout) * time.Minute
	cleanUpInterval := time.Duration(5) * time.Minute
	sessionManager := chariot.NewSessionManager(timeOut, cleanUpInterval)
	if err := vault.InitVaultClient(); err != nil { // Initialize Azure Key Vault client
		cfg.ChariotLogger.Error("Failed to initialize Vault client", zap.Error(err))
		return
	}

	if cfg.ChariotConfig.Headless {
		token := "headless-session"
		userID := "system"
		session := sessionManager.NewSession(userID, slogger, token)
		session.SetOnStart(cfg.ChariotConfig.OnStart) // e.g., "runDecisionService.ch"
		session.SetOnExit(cfg.ChariotConfig.OnExit)   // e.g., "exitDecisionService.ch"
		session.Run()

		// Wait for the session to finish (block main goroutine)
		select {}
	} else {
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

		// Start server with or without SSL
		if cfg.ChariotConfig.SSL {
			fmt.Printf("Starting TLS server on port %d\n", cfg.ChariotConfig.Port)
			e.Logger.Fatal(e.StartTLS(fmt.Sprintf(":%d", cfg.ChariotConfig.Port), fullPathCrt, fullPathKey))
		} else {
			fmt.Printf("Starting HTTP server on port %d (SSL disabled for nginx termination)\n", cfg.ChariotConfig.Port)
			e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", cfg.ChariotConfig.Port)))
		}
	}
}
