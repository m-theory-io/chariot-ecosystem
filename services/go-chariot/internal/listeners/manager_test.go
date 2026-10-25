package listeners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ch "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func testManager(t *testing.T) (*Manager, *ch.Runtime) {
	t.Helper()
	base := t.TempDir()
	rt := ch.NewRuntime()
	ch.RegisterAll(rt)
	m := NewManager(rt)
	m.filePath, m.scriptsDir = filepath.Join(base, "listeners.json"), base
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.Shutdown(ctx); err != nil {
			t.Errorf("supervisor cleanup: %v", err)
		}
		if err := rt.ShutdownListeners(ctx); err != nil {
			t.Errorf("bootstrap cleanup: %v", err)
		}
	})
	return m, rt
}

func script(t *testing.T, m *Manager, name, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(m.scriptsDir, name), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func define(t *testing.T, m *Manager, name, source string) Definition {
	t.Helper()
	def := Definition{Name: name, ScriptFile: name + ".ch"}
	// The current runtime requires explicit, nonzero ports.
	if strings.Contains(source, "listen(0,") {
		source = strings.ReplaceAll(source, "listen(0,", fmt.Sprintf("listen(%d,", unusedPort(t)))
	}
	script(t, m, def.ScriptFile, source)
	if _, err := m.Create(def); err != nil {
		t.Fatal(err)
	}
	return def
}

func listenCode(port int) string {
	return fmt.Sprintf(`listen(%d, func(req) { return("devtest") }, map("bind_address", "127.0.0.1"))`, port)
}

func unusedPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func request(t *testing.T, address string) string {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
	res, err := client.Get("http://" + address + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("HTTP %d: %s", res.StatusCode, body)
	}
	return string(body)
}

func assertClosed(t *testing.T, address string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Fatalf("port still open: %s", address)
	}
}

func stop(t *testing.T, m *Manager, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	l, err := m.Stop(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	if l.Status != "stopped" || l.IsHealthy {
		t.Fatalf("stop status: %+v", l)
	}
}

func TestStartServesHTTPAndStopClosesPortWithoutOwningBootstrap(t *testing.T) {
	m, bootstrap := testManager(t)
	productionPort := unusedPort(t)
	if _, err := bootstrap.ExecProgram(listenCode(productionPort)); err != nil {
		t.Fatal(err)
	}
	if err := bootstrap.SaveFunction("answer", `function answer(req) { return("inherited") }`, ""); err != nil {
		t.Fatal(err)
	}
	define(t, m, "test", `listen(0, "answer", map("bind_address", "127.0.0.1"))`)
	l, err := m.Start("test")
	if err != nil {
		t.Fatal(err)
	}
	if l.Status != "running" || !l.IsHealthy || len(l.Resources) != 1 || l.StartTime.IsZero() {
		t.Fatalf("start status: %+v", l)
	}
	address := l.Resources[0].Address
	if got := request(t, address); got != `"inherited"` {
		t.Fatalf("response: %q", got)
	}
	current, _ := m.Get("test")
	if current.LastActive.IsZero() || current.Resources[0].RequestCount != 1 {
		t.Fatalf("activity not live: %+v", current)
	}
	// Returned snapshots cannot mutate the manager's definition or resource list.
	l.Name = "changed"
	l.Resources[0].State = "changed"
	if again, _ := m.Get("test"); again.Name != "test" || again.Resources[0].State != "running" {
		t.Fatal("aliased snapshot")
	}
	if _, err := m.Start("test"); err != nil {
		t.Fatal(err)
	}
	stop(t, m, "test")
	assertClosed(t, address)
	stop(t, m, "test")
	request(t, fmt.Sprintf("127.0.0.1:%d", productionPort))
	if len(bootstrap.ListListeners()) != 1 {
		t.Fatal("bootstrap ownership changed")
	}
}

func TestStartupRollbackAndZeroResources(t *testing.T) {
	for _, failure := range []string{"script", "hook", "zero", "parse", "missing-hook"} {
		t.Run(failure, func(t *testing.T) {
			m, _ := testManager(t)
			port := unusedPort(t)
			source := listenCode(port)
			if failure == "script" {
				source += "\nundefinedFunction()"
			}
			if failure == "zero" {
				source = `setq(x, 1)`
			}
			if failure == "parse" {
				source = `listen(`
			}
			def := define(t, m, "broken", source)
			if failure == "hook" {
				def.OnStart = "undefinedFunction()"
			}
			if failure == "missing-hook" {
				def.OnExitFile = "absent.ch"
			}
			if _, err := m.Update(def.Name, def); err != nil {
				t.Fatal(err)
			}
			l, err := m.Start(def.Name)
			if err == nil || l.Status != "error" || l.IsHealthy || l.LastError == "" {
				t.Fatalf("startup should fail: %+v, %v", l, err)
			}
			assertClosed(t, fmt.Sprintf("127.0.0.1:%d", port))
			if i := m.instances[def.Name]; i.runtime != nil {
				t.Fatal("failed runtime retained after completed rollback")
			}
		})
	}
}

func TestPortConflictRollsBackEarlierResourcesAndDeleteRunningRejected(t *testing.T) {
	m, _ := testManager(t)
	port, rollbackPort := unusedPort(t), unusedPort(t)
	define(t, m, "first", listenCode(port))
	define(t, m, "second", listenCode(rollbackPort)+"\n"+listenCode(port))
	if _, err := m.Start("first"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete("first"); err == nil {
		t.Fatal("deleted running definition")
	}
	if _, err := m.Update("first", Definition{ScriptFile: "first.ch"}); err == nil {
		t.Fatal("updated running definition")
	}
	if _, err := m.Start("second"); err == nil {
		t.Fatal("conflict should fail synchronously")
	}
	assertClosed(t, fmt.Sprintf("127.0.0.1:%d", rollbackPort))
	request(t, fmt.Sprintf("127.0.0.1:%d", port))
	stop(t, m, "first")
	if err := m.Delete("first"); err != nil {
		t.Fatal(err)
	}
}

func TestV1MigrationResetsLiveStateAndPersistsOnlyV2Config(t *testing.T) {
	m, _ := testManager(t)
	data := `{"version":1,"listeners":{"legacy":{"name":"legacy","script_file":"legacy.ch","auto_start":true,"status":"running","is_healthy":true,"start_time":"2026-01-01T00:00:00Z","last_active":"2026-01-01T00:00:00Z"}}}`
	if err := os.WriteFile(m.filePath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	l, _ := m.Get("legacy")
	if l.Status != "stopped" || l.IsHealthy || !l.StartTime.IsZero() || !l.LastActive.IsZero() || l.AutoStartDevTest {
		t.Fatalf("stale state: %+v", l)
	}
	def := l.Definition
	def.Script, def.AutoStartDevTest = "updated", true
	if _, err := m.Update("legacy", def); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(m.filePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"status"`, `"is_healthy"`, `"start_time"`, `"last_active"`, `"last_error"`, `"auto_start"`, `"resources"`} {
		if strings.Contains(string(saved), forbidden) {
			t.Fatalf("persisted %s: %s", forbidden, saved)
		}
	}
	var snap Snapshot
	if err := json.Unmarshal(saved, &snap); err != nil || snap.Version != 2 {
		t.Fatalf("v2 snapshot: %s, %v", saved, err)
	}
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	l, _ = m.Get("legacy")
	if l.Status != "stopped" || !l.AutoStartDevTest || l.Script != "updated" {
		t.Fatalf("reload: %+v", l)
	}
	if err := m.Delete("legacy"); err != nil {
		t.Fatal(err)
	}
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if len(m.List()) != 0 {
		t.Fatal("delete not persisted")
	}
}

func TestPersistenceFailureRollsBackDefinitionMutations(t *testing.T) {
	m, _ := testManager(t)
	def := define(t, m, "original", listenCode(0))
	m.filePath = m.scriptsDir // Renaming a file over a directory must fail.
	if _, err := m.Create(Definition{Name: "new", ScriptFile: "new.ch"}); err == nil {
		t.Fatal("create should fail")
	}
	if _, err := m.Get("new"); err == nil {
		t.Fatal("failed create was retained")
	}
	changed := def
	changed.Script = "changed"
	if _, err := m.Update("original", changed); err == nil {
		t.Fatal("update should fail")
	}
	if got, _ := m.Get("original"); got.Script != def.Script {
		t.Fatal("failed update was retained")
	}
	if err := m.Delete("original"); err == nil {
		t.Fatal("delete should fail")
	}
	if _, err := m.Get("original"); err != nil {
		t.Fatal("failed delete removed definition")
	}
}

func TestSafeScriptResolution(t *testing.T) {
	m, _ := testManager(t)
	for _, name := range []string{"", "../outside.ch", "/tmp/outside.ch", `sub\file.ch`, "bad.txt", "bad\x00.ch"} {
		if _, err := m.Create(Definition{Name: "bad", ScriptFile: name}); err == nil {
			t.Fatalf("accepted unsafe filename %q", name)
		}
	}
	outside := filepath.Join(t.TempDir(), "outside.ch")
	if err := os.WriteFile(outside, []byte(listenCode(0)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(m.scriptsDir, "escape.ch")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(Definition{Name: "escape", ScriptFile: "escape.ch"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start("escape"); err == nil {
		t.Fatal("followed escaping symlink")
	}
}

func TestLiveStatusDoesNotClaimRunningAfterResourcesStop(t *testing.T) {
	m, _ := testManager(t)
	define(t, m, "live", listenCode(0))
	if _, err := m.Start("live"); err != nil {
		t.Fatal(err)
	}
	rt := m.instances["live"].runtime
	if err := rt.ShutdownListeners(context.Background()); err != nil {
		t.Fatal(err)
	}
	l, _ := m.Get("live")
	if l.Status != "error" || l.IsHealthy {
		t.Fatalf("stale live status: %+v", l)
	}
	if err := m.Delete("live"); err == nil {
		t.Fatal("delete abandoned owned runtime")
	}
	stop(t, m, "live")
}

func TestConcurrentStartStopAndInventoryDoNotHoldManagerLock(t *testing.T) {
	m, rt := testManager(t)
	entered, release := make(chan struct{}), make(chan struct{})
	rt.Register("startupGate", func(...ch.Value) (ch.Value, error) { close(entered); <-release; return ch.Bool(true), nil })
	define(t, m, "gated", "startupGate()\n"+listenCode(0))
	started := make(chan error, 1)
	go func() { _, err := m.Start("gated"); started <- err }()
	<-entered
	// These must remain responsive while interpreter work is blocked.
	if got, _ := m.Get("gated"); got.Status != "starting" {
		t.Fatalf("phase: %+v", got)
	}
	if _, err := m.Start("gated"); err == nil {
		t.Fatal("concurrent start admitted")
	}
	if err := m.Delete("gated"); err == nil {
		t.Fatal("deleted starting definition")
	}
	if _, err := m.Create(Definition{Name: "other", ScriptFile: "other.ch"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.Stop(ctx, "gated")
			if err != nil {
				t.Errorf("concurrent stop: %v", err)
			}
		}()
	}
	close(release)
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 30; n++ {
		m.List()
		_, _ = m.Get("gated")
	}
	wg.Wait()
	if got, _ := m.Get("gated"); got.Status != "stopped" {
		t.Fatalf("final phase: %+v", got)
	}
}

func TestHookFilesAreIsolatedAndResourceHooksAreNotRepeated(t *testing.T) {
	m, bootstrap := testManager(t)
	var startCount, exitCount atomic.Int32
	bootstrap.Register("countStart", func(...ch.Value) (ch.Value, error) { startCount.Add(1); return ch.Bool(true), nil })
	bootstrap.Register("countExit", func(...ch.Value) (ch.Value, error) { exitCount.Add(1); return ch.Bool(true), nil })
	script(t, m, "startHook.ch", `function startHook(port) { countStart() }`)
	script(t, m, "exitHook.ch", `function exitHook(port) { countExit() }`)
	def := define(t, m, "hooks", `listen(0, func(req) { return("ok") }, map("bind_address", "127.0.0.1", "on_start", "startHook", "on_exit", "exitHook"))`)
	def.OnStartFile, def.OnExitFile = "startHook.ch", "exitHook.ch"
	if _, err := m.Update("hooks", def); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start("hooks"); err != nil {
		t.Fatal(err)
	}
	stop(t, m, "hooks")
	if startCount.Load() != 1 || exitCount.Load() != 1 {
		t.Fatalf("duplicate hooks: start=%d exit=%d", startCount.Load(), exitCount.Load())
	}
	if _, ok := bootstrap.GetFunction("startHook"); ok {
		t.Fatal("hook polluted bootstrap library")
	}
	if _, ok := bootstrap.GetFunction("exitHook"); ok {
		t.Fatal("exit hook polluted bootstrap library")
	}
}

func TestStopDeadlineRetainsOwnershipUntilExitHookDrains(t *testing.T) {
	m, bootstrap := testManager(t)
	entered, release := make(chan struct{}), make(chan struct{})
	bootstrap.Register("exitGate", func(...ch.Value) (ch.Value, error) { close(entered); <-release; return ch.Bool(true), nil })
	define(t, m, "draining", `listen(0, func(req) { return("ok") }, map("bind_address", "127.0.0.1", "on_exit", "exitGate()"))`)
	l, err := m.Start("draining")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Stop(ctx, "draining"); !errors.Is(err, context.Canceled) {
		t.Fatalf("deadline: %v", err)
	}
	<-entered
	assertClosed(t, l.Resources[0].Address)
	if got, _ := m.Get("draining"); got.Status != "stopping" {
		t.Fatalf("released ownership: %+v", got)
	}
	if _, err := m.Start("draining"); err == nil {
		t.Fatal("start admitted during disposal")
	}
	if err := m.Delete("draining"); err == nil {
		t.Fatal("deleted draining instance")
	}
	close(release)
	stop(t, m, "draining")
}

func TestShutdownDrainsDevTestOnlyAndPreventsNewStarts(t *testing.T) {
	m, _ := testManager(t)
	define(t, m, "one", listenCode(0))
	l, err := m.Start("one")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertClosed(t, l.Resources[0].Address)
	if _, err := m.Start("one"); err == nil {
		t.Fatal("start admitted after shutdown")
	}
	if err := m.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

type countingCloser struct{ count atomic.Int32 }

func (c *countingCloser) Close() error { c.count.Add(1); return nil }

func TestOwnedHostResourcesCloseButBorrowedBootstrapResourcesDoNot(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprintf("rollback=%v", rollback), func(t *testing.T) {
			m, bootstrap := testManager(t)
			borrowed, owned := &countingCloser{}, &countingCloser{}
			bootstrap.RegisterHostObject("borrowed", borrowed, false)
			bootstrap.Register("createOwned", func(...ch.Value) (ch.Value, error) {
				m.mu.RLock()
				rt := m.instances["owned"].runtime
				m.mu.RUnlock()
				rt.RegisterHostObject("owned", owned, false)
				return ch.Bool(true), nil
			})
			source := "createOwned()\n" + listenCode(0)
			if rollback {
				source += "\nundefinedFunction()"
			}
			define(t, m, "owned", source)
			_, err := m.Start("owned")
			if rollback && err == nil {
				t.Fatal("expected startup failure")
			}
			if !rollback {
				if err != nil {
					t.Fatal(err)
				}
				stop(t, m, "owned")
			}
			if owned.count.Load() != 1 || borrowed.count.Load() != 0 {
				t.Fatalf("resource ownership: owned=%d borrowed=%d", owned.count.Load(), borrowed.count.Load())
			}
		})
	}
}

func TestRegistryHooksRunInOrderAndExitErrorsStillCloseAllResources(t *testing.T) {
	m, bootstrap := testManager(t)
	var stages []string
	bootstrap.Register("stage", func(args ...ch.Value) (ch.Value, error) {
		stages = append(stages, string(args[0].(ch.Str)))
		return ch.Bool(true), nil
	})
	def := define(t, m, "ordered", `stage("primary")
listen(0, func(req) { return("ok") }, map("bind_address", "127.0.0.1", "on_exit", "stage(\"resource-exit\")"))`)
	def.OnStart, def.OnExit = `stage("registry-start")`, `stage("registry-exit") undefinedFunction()`
	if _, err := m.Update("ordered", def); err != nil {
		t.Fatal(err)
	}
	l, err := m.Start("ordered")
	if err != nil {
		t.Fatal(err)
	}
	address := l.Resources[0].Address
	l, err = m.Stop(context.Background(), "ordered")
	if err == nil || l.Status != "stopped" || l.LastError == "" {
		t.Fatalf("exit error lost: %+v, %v", l, err)
	}
	assertClosed(t, address)
	if got := strings.Join(stages, ","); got != "primary,registry-start,resource-exit,registry-exit" {
		t.Fatalf("hook order: %s", got)
	}
	if len(m.instances) != 0 {
		t.Fatal("runtime retained after completed disposal")
	}
}
