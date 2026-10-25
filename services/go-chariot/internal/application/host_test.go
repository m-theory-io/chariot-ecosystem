package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/logs"
)

func newHostTestRuntime(t *testing.T, listenerPort int) *ApplicationRuntime {
	t.Helper()
	runtime := chariot.NewRuntime()
	chariot.RegisterAll(runtime)
	if err := runtime.SaveFunction("applicationHandler", `function applicationHandler(request) { return(applicationMarker) }`, ""); err != nil {
		t.Fatalf("register application handler: %v", err)
	}
	source := fmt.Sprintf(`
declareGlobal(applicationMarker, 'S', 'shared')
listen(%d, "applicationHandler", map("bind_address", "127.0.0.1"))
`, listenerPort)
	if _, err := runtime.ExecProgram(source); err != nil {
		t.Fatalf("initialize test runtime: %v", err)
	}
	return &ApplicationRuntime{Runtime: runtime, BootstrapLoaded: true}
}

func waitForPort(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("port %d did not open: %v", port, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHostUsesOneRuntimeAndGracefullyStopsListeners(t *testing.T) {
	originalDataPath := cfg.ChariotConfig.DataPath
	originalListenersFile := cfg.ChariotConfig.ListenersFile
	cfg.ChariotConfig.DataPath = t.TempDir()
	cfg.ChariotConfig.ListenersFile = "listeners.json"
	t.Cleanup(func() {
		cfg.ChariotConfig.DataPath = originalDataPath
		cfg.ChariotConfig.ListenersFile = originalListenersFile
	})

	for _, devREST := range []bool{false, true} {
		t.Run(fmt.Sprintf("dev_rest_%t", devREST), func(t *testing.T) {
			listenerPort := availablePort(t)
			applicationRuntime := newHostTestRuntime(t, listenerPort)
			sessionManager := chariot.NewSessionManager(time.Minute, time.Minute)
			logger := logs.NewZapLogger()
			defer logger.Sync()
			restPort := availablePort(t)
			host, err := NewHost(HostOptions{
				Runtime: applicationRuntime, SessionManager: sessionManager, Logger: logger,
				DevRESTEnabled: devREST, Port: restPort, ShutdownTimeout: 2 * time.Second,
			})
			if err != nil {
				t.Fatalf("create host: %v", err)
			}

			// Handler/session construction must not execute bootstrap and bind a second listener.
			if statuses := applicationRuntime.Runtime.ListListeners(); len(statuses) != 1 {
				t.Fatalf("listener count = %d, want one bootstrap listener", len(statuses))
			}
			session := sessionManager.NewSession("test-user", logger, "test-token")
			result, err := session.Runtime.ExecProgram(`applicationHandler(map())`)
			if err != nil || result != chariot.Str("shared") {
				t.Fatalf("session did not inherit application function/state: result=%v err=%v", result, err)
			}
			if statuses := applicationRuntime.Runtime.ListListeners(); len(statuses) != 1 {
				t.Fatalf("session creation changed listener ownership: %d", len(statuses))
			}
			sessionRuntime := session.Runtime
			if _, err := sessionRuntime.ExecProgram(fmt.Sprintf(`listen(%d, "applicationHandler", map("bind_address", "127.0.0.1"))`, availablePort(t))); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			runDone := make(chan error, 1)
			go func() { runDone <- host.Run(ctx) }()
			if devREST {
				waitForPort(t, restPort)
			}
			waitForPort(t, listenerPort)
			cancel()
			if err := <-runDone; err != nil {
				t.Fatalf("run host: %v", err)
			}
			if statuses := applicationRuntime.Runtime.ListListeners(); len(statuses) != 0 {
				t.Fatalf("listeners remained after shutdown: %+v", statuses)
			}
			if len(sessionRuntime.ListListeners()) != 0 || sessionManager.GetActiveSessions() != 0 {
				t.Fatal("host shutdown leaked session listeners")
			}
		})
	}
}

func TestHostShutdownClosesAPICreatedDevTestAndApplicationListeners(t *testing.T) {
	originalDataPath := cfg.ChariotConfig.DataPath
	originalListenersFile := cfg.ChariotConfig.ListenersFile
	originalScriptsDir := cfg.ChariotConfig.ListenerScriptsDir
	originalFunctionLib := cfg.ChariotConfig.FunctionLib
	cfg.ChariotConfig.DataPath = t.TempDir()
	cfg.ChariotConfig.ListenersFile = "listeners.json"
	cfg.ChariotConfig.ListenerScriptsDir = ""
	cfg.ChariotConfig.FunctionLib = ""
	t.Cleanup(func() {
		cfg.ChariotConfig.DataPath = originalDataPath
		cfg.ChariotConfig.ListenersFile = originalListenersFile
		cfg.ChariotConfig.ListenerScriptsDir = originalScriptsDir
		cfg.ChariotConfig.FunctionLib = originalFunctionLib
	})

	applicationPort := availablePort(t)
	applicationRuntime := newHostTestRuntime(t, applicationPort)
	sessionManager := chariot.NewSessionManager(time.Minute, time.Minute)
	logger := logs.NewZapLogger()
	defer logger.Sync()
	host, err := NewHost(HostOptions{
		Runtime: applicationRuntime, SessionManager: sessionManager, Logger: logger,
		DevRESTEnabled: true, ShutdownTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := host.Shutdown(ctx); err != nil {
			t.Errorf("cleanup host: %v", err)
		}
	})
	sessionManager.NewSession("host-integration", logger, "host-integration-token")

	// Use the real router, authentication middleware, and handlers, not direct
	// supervisor calls. The script editor and registry have distinct payloads.
	api := func(method, path string, payload any) json.RawMessage {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(method, path, strings.NewReader(string(body)))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer host-integration-token")
		response := httptest.NewRecorder()
		host.Echo().ServeHTTP(response, request)
		var result struct {
			Result string          `json:"result"`
			Data   json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("%s %s: decode response: %v; body=%s", method, path, err, response.Body.String())
		}
		if response.Code != http.StatusOK || result.Result != "OK" {
			t.Fatalf("%s %s: status=%d body=%s", method, path, response.Code, response.Body.String())
		}
		return result.Data
	}
	devTestPort := availablePort(t)
	api(http.MethodPost, "/api/listener-scripts", map[string]string{
		"name":    "host-devtest.ch",
		"content": fmt.Sprintf(`listen(%d, "applicationHandler", map("bind_address", "127.0.0.1"))`, devTestPort),
	})
	api(http.MethodPost, "/api/listeners", map[string]any{
		"name": "host-devtest", "script_file": "host-devtest.ch", "auto_start_devtest": false,
	})
	api(http.MethodPost, "/api/listeners/host-devtest/start", nil)

	client := &http.Client{Timeout: time.Second}
	t.Cleanup(client.CloseIdleConnections)
	for _, port := range []int{applicationPort, devTestPort} {
		response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/", port))
		if err != nil {
			t.Fatalf("request listener %d: %v", port, err)
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusOK || !strings.Contains(string(body), "shared") {
			t.Fatalf("listener %d: status=%d body=%s err=%v", port, response.StatusCode, body, readErr)
		}
	}
	if statuses := applicationRuntime.Runtime.ListListeners(); len(statuses) != 1 || statuses[0].Port != applicationPort {
		t.Fatalf("DevTest must be isolated from application listener ownership: %+v", statuses)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := host.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown host: %v", err)
	}
	for _, port := range []int{devTestPort, applicationPort} {
		address := fmt.Sprintf("127.0.0.1:%d", port)
		if connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond); err == nil {
			_ = connection.Close()
			t.Errorf("listener %d still accepts connections after host shutdown", port)
		}
		listener, err := net.Listen("tcp", address)
		if err != nil {
			t.Errorf("listener %d was not released: %v", port, err)
		} else {
			_ = listener.Close()
		}
	}
	if len(applicationRuntime.Runtime.ListListeners()) != 0 || sessionManager.GetActiveSessions() != 0 {
		t.Fatal("host shutdown retained application listeners or sessions")
	}
}

func TestConcurrentHostShutdownWaiterHonorsOwnDeadline(t *testing.T) {
	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	rt.Register("blockedExit", func(...chariot.Value) (chariot.Value, error) {
		close(entered)
		<-release
		return chariot.DBNull, nil
	})
	if _, err := rt.ExecProgram(fmt.Sprintf(`listen(%d, func(req) { return(req) }, map("on_exit", "blockedExit()"))`, availablePort(t))); err != nil {
		t.Fatal(err)
	}
	sm := chariot.NewSessionManager(time.Minute, time.Minute)
	logger := logs.NewZapLogger()
	defer logger.Sync()
	host, err := NewHost(HostOptions{Runtime: &ApplicationRuntime{Runtime: rt}, SessionManager: sm, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	first := make(chan error, 1)
	go func() { first <- host.Shutdown(ctx) }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("exit hook did not start")
	}
	short, cancelShort := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelShort()
	if err := host.Shutdown(short); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("concurrent shutdown ignored waiter's deadline: %v", err)
	}
	unblock()
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := host.Shutdown(context.Background()); err != nil {
		t.Fatalf("completed host shutdown not idempotent: %v", err)
	}
}

func TestStartupRetriesIncompleteRollbackWithBoundedFreshCleanup(t *testing.T) {
	// Exercise main without extending ownership to a cmd test file. Build once
	// outside the measured startup interval so compiler latency cannot mask an
	// unbounded cleanup wait. Blocked script goroutines die with the child.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "go-chariot")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../cmd")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build startup test binary: %v\n%s", err, output)
	}
	for _, test := range []struct {
		name        string
		exitDelayMS int
	}{
		{name: "finishes_during_fresh_grace", exitDelayMS: 1500},
		{name: "blocked_script_exceeds_both_budgets", exitDelayMS: 60000},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			port := availablePort(t)
			bootstrap := fmt.Sprintf(`
listen(%d, func(req) { return(req) }, map("bind_address", "127.0.0.1", "on_exit", "sleep(%d)"))
missingBootstrapFunction()
`, port, test.exitDelayMS)
			if err := os.WriteFile(filepath.Join(directory, "bootstrap.ch"), []byte(bootstrap), 0o600); err != nil {
				t.Fatal(err)
			}
			secrets := filepath.Join(directory, "secrets.json")
			if err := os.WriteFile(secrets, []byte(`{}`), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "functions.json"), []byte(`{}`), 0o600); err != nil {
				t.Fatal(err)
			}
			startupContext, cancelStartup := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelStartup()
			command := exec.CommandContext(startupContext, binary)
			command.Dir = directory
			// Do not inherit developer credentials, paths, or enabled transports.
			for _, variable := range os.Environ() {
				if !strings.HasPrefix(variable, "CHARIOT_") {
					command.Env = append(command.Env, variable)
				}
			}
			command.Env = append(command.Env,
				"CHARIOT_HEADLESS=true", "CHARIOT_DEV_REST_ENABLED=false", "CHARIOT_MCP_ENABLED=false",
				"CHARIOT_SECRET_PROVIDER=file", "CHARIOT_SECRET_FILE_PATH="+secrets,
				"CHARIOT_DATA_PATH="+directory, "CHARIOT_BOOTSTRAP=bootstrap.ch",
				"CHARIOT_TREE_PATH="+directory, "CHARIOT_FUNCTION_LIB=functions.json",
				"CHARIOT_SHUTDOWN_TIMEOUT=1", "CHARIOT_NSQ_ENABLED=false", "CHARIOT_NSQ_RESPONSE_TOPICS=",
			)
			started := time.Now()
			output, err := command.CombinedOutput()
			elapsed := time.Since(started)
			var exitError *exec.ExitError
			if startupContext.Err() != nil || !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
				t.Fatalf("startup did not exit with bounded failure: elapsed=%v err=%v context=%v\n%s", elapsed, err, startupContext.Err(), output)
			}
			for _, diagnostic := range []string{"initialize application runtime", "missingBootstrapFunction", "rollback listeners", "cleanup application runtime", "deadline exceeded"} {
				if !strings.Contains(string(output), diagnostic) {
					t.Errorf("missing joined startup diagnostic %q:\n%s", diagnostic, output)
				}
			}
			// The original rollback gets one second; the caller must actually
			// retain the returned runtime and wait under a new cleanup deadline.
			minimum := 1400 * time.Millisecond
			if test.exitDelayMS > 2000 {
				minimum = 1900 * time.Millisecond
			}
			if elapsed < minimum {
				t.Errorf("startup ignored the retained runtime: elapsed=%v, want at least %v", elapsed, minimum)
			}
			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				t.Fatalf("failed startup left listener port bound: %v", err)
			}
			_ = listener.Close()
		})
	}
}

func TestHostReturnsDevRESTBindFailure(t *testing.T) {
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("occupy REST port: %v", err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	runtime := chariot.NewRuntime()
	chariot.RegisterAll(runtime)
	sessionManager := chariot.NewSessionManager(time.Minute, time.Minute)
	logger := logs.NewZapLogger()
	defer logger.Sync()
	host, err := NewHost(HostOptions{
		Runtime: &ApplicationRuntime{Runtime: runtime}, SessionManager: sessionManager, Logger: logger,
		DevRESTEnabled: true, Port: port, ShutdownTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("create host: %v", err)
	}

	err = host.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "Dev REST server") {
		t.Fatalf("run error = %v, want REST bind failure", err)
	}
}
