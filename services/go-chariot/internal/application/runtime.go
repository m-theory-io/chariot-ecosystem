package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

// RuntimeOptions describes the files used to initialize the application runtime.
type RuntimeOptions struct {
	FunctionLibrary string
	Bootstrap       string
	RollbackTimeout time.Duration // Defaults to configured shutdown_timeout, then 30s.
}

// ApplicationRuntime is the single initialized runtime owned by an application host.
type ApplicationRuntime struct {
	Runtime         *chariot.Runtime
	BootstrapLoaded bool
}

// BuildApplicationRuntime creates and transactionally initializes one application runtime.
// On rollback timeout it may return both an error and a non-nil runtime that
// still owns live listeners. Callers must retain that runtime and await/retry
// ShutdownListeners; an error does not imply that arbitrary scripts have exited.
func BuildApplicationRuntime(ctx context.Context, options RuntimeOptions) (*ApplicationRuntime, error) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)

	rollback := func(initializationErr error) (*ApplicationRuntime, error) {
		grace := options.RollbackTimeout
		if grace <= 0 {
			grace = time.Duration(cfg.ChariotConfig.ShutdownTimeout) * time.Second
			if grace <= 0 {
				grace = 30 * time.Second
			}
		}
		rollbackContext, cancel := context.WithTimeout(context.Background(), grace)
		defer cancel()
		shutdownErr := rt.ShutdownListeners(rollbackContext)
		if shutdownErr != nil {
			// Return the failed runtime when live resources still require ownership.
			var pending *ApplicationRuntime
			if len(rt.ListListeners()) != 0 {
				pending = &ApplicationRuntime{Runtime: rt}
			}
			return pending, errors.Join(initializationErr, fmt.Errorf("rollback listeners: %w", shutdownErr))
		}
		return nil, initializationErr
	}
	if err := ctx.Err(); err != nil {
		return rollback(fmt.Errorf("initialize application runtime: %w", err))
	}

	if options.FunctionLibrary != "" {
		functions, err := chariot.LoadFunctionsFromFile(options.FunctionLibrary)
		if err != nil {
			return rollback(fmt.Errorf("load function library %q: %w", options.FunctionLibrary, err))
		}
		for name, function := range functions {
			rt.RegisterFunction(name, function)
		}
	}

	applicationRuntime := &ApplicationRuntime{Runtime: rt}
	if options.Bootstrap == "" {
		return applicationRuntime, nil
	}

	bootstrapPath, err := chariot.GetSecureFilePath(options.Bootstrap, "data")
	if err != nil {
		return rollback(fmt.Errorf("resolve bootstrap %q: %w", options.Bootstrap, err))
	}
	bootstrap, err := os.ReadFile(bootstrapPath)
	if err != nil {
		return rollback(fmt.Errorf("read bootstrap %q: %w", bootstrapPath, err))
	}
	if err := ctx.Err(); err != nil {
		return rollback(fmt.Errorf("execute bootstrap %q: %w", bootstrapPath, err))
	}
	if _, err := rt.ExecProgram(string(bootstrap)); err != nil {
		return rollback(fmt.Errorf("execute bootstrap %q: %w", bootstrapPath, err))
	}
	applicationRuntime.BootstrapLoaded = true
	return applicationRuntime, nil
}
