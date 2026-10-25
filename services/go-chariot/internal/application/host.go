package application

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/handlers"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/internal/routes"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
	mcpserver "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/mcp"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

// HostOptions configures optional process surfaces around one application runtime.
type HostOptions struct {
	Runtime         *ApplicationRuntime
	SessionManager  *chariot.SessionManager
	Logger          *logs.ZapLogger
	DevRESTEnabled  bool
	Port            int
	SSL             bool
	CertPath        string
	MCPEnabled      bool
	MCPTransport    string
	MCPPath         string
	ShutdownTimeout time.Duration
}

// Host owns the process-level REST, session, and application-runtime lifecycle.
type Host struct {
	options      HostOptions
	echo         *echo.Echo
	handlers     *handlers.Handlers
	shutdownOnce sync.Once
	shutdownDone chan struct{}
	shutdownErr  error
}

// NewHost creates process surfaces without starting them.
func NewHost(options HostOptions) (*Host, error) {
	if options.Runtime == nil || options.Runtime.Runtime == nil {
		return nil, errors.New("application runtime is required")
	}
	if options.SessionManager == nil {
		return nil, errors.New("session manager is required")
	}
	if options.Logger == nil {
		return nil, errors.New("logger is required")
	}
	if options.ShutdownTimeout <= 0 {
		options.ShutdownTimeout = 30 * time.Second
	}

	host := &Host{options: options, shutdownDone: make(chan struct{})}
	options.SessionManager.SetBootstrapRuntime(options.Runtime.Runtime)
	if options.DevRESTEnabled {
		host.echo = host.newEcho()
	}
	return host, nil
}

func (h *Host) newEcho() *echo.Echo {
	e := echo.New()
	handlers := handlers.NewHandlers(h.options.SessionManager, h.options.Runtime.Runtime)
	h.handlers = handlers
	routes.RegisterRoutes(e, handlers)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(logs.ZapLoggerMiddleware(h.options.Logger.Get()))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	if h.options.MCPEnabled {
		mcpOptions := mcpserver.ServerOptions{RuntimeProvider: func() *chariot.Runtime {
			return h.options.Runtime.Runtime
		}}
		switch strings.ToLower(h.options.MCPTransport) {
		case "ws":
			e.GET(h.options.MCPPath, func(c echo.Context) error {
				return mcpserver.HandleWSWithOptions(c, mcpOptions)
			})
			h.options.Logger.Info("MCP WebSocket route enabled", zap.String("path", h.options.MCPPath))
		case "http", "sse":
			handler := echo.WrapHandler(mcpserver.NewHTTPHandler(mcpOptions))
			e.GET(h.options.MCPPath, handler)
			e.POST(h.options.MCPPath, handler)
			h.options.Logger.Info("MCP HTTP/SSE route enabled", zap.String("path", h.options.MCPPath))
		}
	}
	return e
}

// Run starts optional REST service and blocks until cancellation or server failure.
func (h *Host) Run(ctx context.Context) error {
	var serverErrors <-chan error
	if h.echo != nil {
		errorsChannel := make(chan error, 1)
		serverErrors = errorsChannel
		address := fmt.Sprintf(":%d", h.options.Port)
		go func() {
			var err error
			if h.options.SSL {
				err = h.echo.StartTLS(address,
					filepath.Join(h.options.CertPath, "server.crt"),
					filepath.Join(h.options.CertPath, "server.key"))
			} else {
				err = h.echo.Start(address)
			}
			errorsChannel <- err
		}()
	}

	var runErr error
	if serverErrors == nil {
		<-ctx.Done()
	} else {
		select {
		case <-ctx.Done():
		case err := <-serverErrors:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				runErr = fmt.Errorf("Dev REST server: %w", err)
			}
		}
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), h.options.ShutdownTimeout)
	defer cancel()
	if err := h.Shutdown(shutdownContext); err != nil {
		return errors.Join(runErr, fmt.Errorf("application shutdown: %w", err))
	}
	return runErr
}

// Shutdown stops accepting REST work before stopping runtime listeners and sessions.
func (h *Host) Shutdown(ctx context.Context) error {
	h.shutdownOnce.Do(func() {
		go func() {
			defer close(h.shutdownDone)
			var shutdownErrors []error
			if h.echo != nil {
				if err := h.echo.Shutdown(ctx); err != nil {
					shutdownErrors = append(shutdownErrors, fmt.Errorf("stop Dev REST: %w", err))
					_ = h.echo.Close()
				}
			}
			if h.handlers != nil {
				if err := h.handlers.ShutdownDevTest(ctx); err != nil {
					shutdownErrors = append(shutdownErrors, fmt.Errorf("stop DevTest listeners: %w", err))
				}
			}
			if err := h.options.SessionManager.Shutdown(ctx); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("stop sessions: %w", err))
			}
			if err := h.options.Runtime.Runtime.ShutdownListeners(ctx); err != nil {
				shutdownErrors = append(shutdownErrors, fmt.Errorf("stop runtime listeners: %w", err))
			}
			h.shutdownErr = errors.Join(shutdownErrors...)
		}()
	})
	select {
	case <-h.shutdownDone:
		return h.shutdownErr
	default:
	}
	select {
	case <-h.shutdownDone:
		return h.shutdownErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Echo returns the optional Dev REST server for integration testing.
func (h *Host) Echo() *echo.Echo {
	return h.echo
}
