package config

import (
	"os"
	"path/filepath"
	"testing"
)

func stubConfig(t *testing.T) {
	t.Helper()
	original := *ChariotConfig
	t.Cleanup(func() {
		*ChariotConfig = original
	})
}

func TestSanitizeSandboxKey(t *testing.T) {
	cases := map[string]string{
		"":                         "",
		"User.Name+1@example.com":  "user-name1-example-com",
		"   spaces   ":             "spaces",
		"Already-Clean":            "already-clean",
		"double__delimiters@@test": "double-delimiters-test",
	}
	for input, want := range cases {
		if got := SanitizeSandboxKey(input); got != want {
			t.Fatalf("sanitize %q: got %q want %q", input, got, want)
		}
	}
}

func TestDefaultStorageScope(t *testing.T) {
	stubConfig(t)
	ChariotConfig.SandboxEnabled = true
	ChariotConfig.SandboxDefaultScope = ""
	if got := DefaultStorageScope(); got != StorageScopeSandbox {
		t.Fatalf("expected sandbox default when enabled, got %q", got)
	}
	ChariotConfig.SandboxDefaultScope = "GLOBAL"
	if got := DefaultStorageScope(); got != StorageScopeGlobal {
		t.Fatalf("expected global default when configured, got %q", got)
	}
	ChariotConfig.SandboxEnabled = false
	ChariotConfig.SandboxDefaultScope = "sandbox"
	if got := DefaultStorageScope(); got != StorageScopeGlobal {
		t.Fatalf("sandbox disabled should force global, got %q", got)
	}
}

func TestResolveStorageScopeFallback(t *testing.T) {
	stubConfig(t)
	ChariotConfig.SandboxEnabled = false
	if got := ResolveStorageScope("sandbox"); got != StorageScopeGlobal {
		t.Fatalf("disabled sandbox should resolve to global, got %q", got)
	}
	ChariotConfig.SandboxEnabled = true
	ChariotConfig.SandboxDefaultScope = "global"
	if got := ResolveStorageScope(" "); got != StorageScopeGlobal {
		t.Fatalf("empty scope should fall back to configured default, got %q", got)
	}
}

func TestEnsureStorageBaseSandbox(t *testing.T) {
	stubConfig(t)
	root := t.TempDir()
	ChariotConfig.SandboxEnabled = true
	ChariotConfig.SandboxRoot = filepath.Join(root, "sandboxes")

	base, err := EnsureStorageBase(StorageKindDiagram, StorageScopeSandbox, "User.Name+1@example.com")
	if err != nil {
		t.Fatalf("EnsureStorageBase sandbox: %v", err)
	}
	want := filepath.Join(ChariotConfig.SandboxRoot, "user-name1-example-com", "diagrams")
	if base != want {
		t.Fatalf("EnsureStorageBase sandbox path mismatch: got %q want %q", base, want)
	}
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		t.Fatalf("sandbox directory not created: %v", err)
	}
}

func TestEnsureStorageBaseGlobalFallback(t *testing.T) {
	stubConfig(t)
	root := t.TempDir()
	ChariotConfig.SandboxEnabled = false
	ChariotConfig.DiagramPath = filepath.Join(root, "diagrams")

	base, err := EnsureStorageBase(StorageKindDiagram, StorageScopeSandbox, "user")
	if err != nil {
		t.Fatalf("EnsureStorageBase global fallback: %v", err)
	}
	if base != ChariotConfig.DiagramPath {
		t.Fatalf("expected global diagram path, got %q", base)
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("global diagram directory missing: %v", err)
	}
}

func TestEnsureSandboxDirectories(t *testing.T) {
	stubConfig(t)
	root := t.TempDir()

	// Test with sandboxes disabled - should not error
	ChariotConfig.SandboxEnabled = false
	if err := EnsureSandboxDirectories("testuser"); err != nil {
		t.Fatalf("EnsureSandboxDirectories() with disabled should not error, got %v", err)
	}

	// Test with sandboxes enabled
	ChariotConfig.SandboxEnabled = true
	ChariotConfig.SandboxRoot = filepath.Join(root, "sandboxes")

	username := "test@example.com"
	if err := EnsureSandboxDirectories(username); err != nil {
		t.Fatalf("EnsureSandboxDirectories() error = %v", err)
	}

	// Verify all directories were created
	key := SanitizeSandboxKey(username)
	for _, subdir := range []string{"data", "trees", "diagrams"} {
		path := filepath.Join(ChariotConfig.SandboxRoot, key, subdir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("Expected directory %s was not created: %v", path, err)
		}
	}

	// Test idempotence - should not error on second call
	if err := EnsureSandboxDirectories(username); err != nil {
		t.Errorf("EnsureSandboxDirectories() second call error = %v", err)
	}

	// Test with empty username
	if err := EnsureSandboxDirectories(""); err == nil {
		t.Error("EnsureSandboxDirectories() with empty username should error")
	}
}
