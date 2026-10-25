package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
)

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	return port
}

func writeBootstrap(t *testing.T, source string) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, "bootstrap.ch")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write bootstrap: %v", err)
	}
	originalDataPath := cfg.ChariotConfig.DataPath
	cfg.ChariotConfig.DataPath = directory
	t.Cleanup(func() { cfg.ChariotConfig.DataPath = originalDataPath })
	return path
}

func withoutConfiguredFunctionLibrary(t *testing.T) {
	t.Helper()
	original := cfg.ChariotConfig.FunctionLib
	cfg.ChariotConfig.FunctionLib = ""
	t.Cleanup(func() { cfg.ChariotConfig.FunctionLib = original })
}

func TestBuildApplicationRuntimeExecutesBootstrap(t *testing.T) {
	withoutConfiguredFunctionLibrary(t)
	bootstrap := writeBootstrap(t, `declareGlobal(applicationMarker, 'S', 'initialized')`)

	applicationRuntime, err := BuildApplicationRuntime(context.Background(), RuntimeOptions{Bootstrap: bootstrap})
	if err != nil {
		t.Fatalf("build application runtime: %v", err)
	}
	defer applicationRuntime.Runtime.ShutdownListeners(context.Background())
	if !applicationRuntime.BootstrapLoaded {
		t.Fatal("bootstrap was not marked loaded")
	}
	result, err := applicationRuntime.Runtime.ExecProgram(`applicationMarker`)
	if err != nil {
		t.Fatalf("read bootstrap global: %v", err)
	}
	if result != chariot.Str("initialized") {
		t.Fatalf("applicationMarker = %v, want initialized", result)
	}
}

func TestBuildApplicationRuntimeRollsBackListenersOnFailure(t *testing.T) {
	withoutConfiguredFunctionLibrary(t)
	port := availablePort(t)
	bootstrap := writeBootstrap(t, fmt.Sprintf(`
setq(bootstrapHandler, func(request) { return(request) })
listen(%d, bootstrapHandler, map("bind_address", "127.0.0.1"))
missingBootstrapFunction()
`, port))

	_, err := BuildApplicationRuntime(context.Background(), RuntimeOptions{Bootstrap: bootstrap})
	if err == nil || !strings.Contains(err.Error(), "execute bootstrap") {
		t.Fatalf("build error = %v, want bootstrap execution failure", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		listener, listenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if listenErr == nil {
			_ = listener.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("bootstrap listener was not rolled back: %v", listenErr)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBuildApplicationRuntimeRejectsMissingInitializationFiles(t *testing.T) {
	_, err := BuildApplicationRuntime(context.Background(), RuntimeOptions{FunctionLibrary: filepath.Join(t.TempDir(), "missing.json")})
	if err == nil || !strings.Contains(err.Error(), "load function library") {
		t.Fatalf("error = %v, want missing function library failure", err)
	}

	directory := t.TempDir()
	originalDataPath := cfg.ChariotConfig.DataPath
	cfg.ChariotConfig.DataPath = directory
	t.Cleanup(func() { cfg.ChariotConfig.DataPath = originalDataPath })
	_, err = BuildApplicationRuntime(context.Background(), RuntimeOptions{Bootstrap: filepath.Join(directory, "missing.ch")})
	if err == nil || !strings.Contains(err.Error(), "read bootstrap") {
		t.Fatalf("error = %v, want missing bootstrap failure", err)
	}
}

func TestBuildApplicationRuntimeResolvesFunctionLibraryOnce(t *testing.T) {
	directory := t.TempDir()
	libraryName := "functions.json"
	if err := os.WriteFile(filepath.Join(directory, libraryName), []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write function library: %v", err)
	}
	originalTreePath := cfg.ChariotConfig.TreePath
	cfg.ChariotConfig.TreePath = directory
	t.Cleanup(func() { cfg.ChariotConfig.TreePath = originalTreePath })

	applicationRuntime, err := BuildApplicationRuntime(context.Background(), RuntimeOptions{FunctionLibrary: libraryName})
	if err != nil {
		t.Fatalf("build application runtime: %v", err)
	}
	if applicationRuntime == nil || applicationRuntime.Runtime == nil {
		t.Fatal("application runtime was not created")
	}
}

func TestBootstrapRollbackUsesFreshGraceAfterParentCancellation(t *testing.T) {
	withoutConfiguredFunctionLibrary(t)
	port := availablePort(t)
	bootstrap := writeBootstrap(t, fmt.Sprintf(`
listen(%d, func(req) { return(req) }, map("bind_address", "127.0.0.1", "on_exit", "sleep(60)"))
sleep(40)
missingBootstrapFunction()
`, port))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	failed, err := BuildApplicationRuntime(ctx, RuntimeOptions{Bootstrap: bootstrap, RollbackTimeout: time.Second})
	if err == nil || errors.Is(err, context.DeadlineExceeded) || failed != nil {
		t.Fatalf("rollback inherited expired parent instead of fresh grace: runtime=%v err=%v", failed, err)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("rollback left port bound: %v", err)
	}
	_ = listener.Close()
}

func TestBootstrapRollbackIsBoundedWithUnboundedParent(t *testing.T) {
	withoutConfiguredFunctionLibrary(t)
	bootstrap := writeBootstrap(t, fmt.Sprintf(`
listen(%d, func(req) { return(req) }, map("bind_address", "127.0.0.1", "on_exit", "sleep(300)"))
missingBootstrapFunction()
`, availablePort(t)))
	started := time.Now()
	failed, err := BuildApplicationRuntime(context.Background(), RuntimeOptions{Bootstrap: bootstrap, RollbackTimeout: 25 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 250*time.Millisecond {
		t.Fatalf("rollback was not bounded: elapsed=%v err=%v", time.Since(started), err)
	}
	if failed == nil || len(failed.Runtime.ListListeners()) != 1 {
		t.Fatal("failed live runtime not returned for continued ownership")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// The original timeout remains terminal, but a retry waits for real cleanup.
	_ = failed.Runtime.ShutdownListeners(ctx)
	if len(failed.Runtime.ListListeners()) != 0 {
		t.Fatal("rollback listener never finalized")
	}
}
