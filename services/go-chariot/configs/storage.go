package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// StorageScope represents whether a request targets the shared/global space or a sandbox.
type StorageScope string

const (
	StorageScopeGlobal  StorageScope = "global"
	StorageScopeSandbox StorageScope = "sandbox"
)

// StorageKind identifies the logical type of artifact stored on disk.
type StorageKind string

const (
	StorageKindData    StorageKind = "data"
	StorageKindTree    StorageKind = "tree"
	StorageKindDiagram StorageKind = "diagram"
)

// ParseStorageScope parses a caller-provided scope string without applying defaults.
func ParseStorageScope(raw string) StorageScope {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sandbox", "private", "user":
		return StorageScopeSandbox
	case "global", "shared", "public":
		return StorageScopeGlobal
	default:
		return ""
	}
}

// SanitizeSandboxKey converts a username/email into a filesystem-friendly key.
func SanitizeSandboxKey(input string) string {
	cleaned := strings.ToLower(strings.TrimSpace(input))
	if cleaned == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range cleaned {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		switch r {
		case '@', '.', '-', '_':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	key := strings.Trim(b.String(), "-")
	return key
}

// DefaultStorageScope returns the configured default scope, honoring sandbox enablement.
func DefaultStorageScope() StorageScope {
	desired := ParseStorageScope(ChariotConfig.SandboxDefaultScope)
	if desired == StorageScopeSandbox && !ChariotConfig.SandboxEnabled {
		return StorageScopeGlobal
	}
	if desired == "" {
		if ChariotConfig.SandboxEnabled {
			return StorageScopeSandbox
		}
		return StorageScopeGlobal
	}
	return desired
}

// ResolveStorageScope applies configuration defaults to an optional scope hint.
func ResolveStorageScope(raw string) StorageScope {
	scope := ParseStorageScope(raw)
	if scope == "" {
		scope = DefaultStorageScope()
	}
	if scope == StorageScopeSandbox && !ChariotConfig.SandboxEnabled {
		return StorageScopeGlobal
	}
	return scope
}

// EnsureStorageBase resolves and creates (if needed) the directory for the given kind/scope.
func EnsureStorageBase(kind StorageKind, scope StorageScope, username string) (string, error) {
	base, err := storageBasePath(kind, scope, username)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", fmt.Errorf("create storage directory: %w", err)
	}
	return base, nil
}

func storageBasePath(kind StorageKind, scope StorageScope, username string) (string, error) {
	switch scope {
	case StorageScopeGlobal:
		return globalKindPath(kind)
	case StorageScopeSandbox:
		if !ChariotConfig.SandboxEnabled {
			return globalKindPath(kind)
		}
		key := SanitizeSandboxKey(username)
		if key == "" {
			return "", errors.New("sandbox scope requires authenticated username")
		}
		subdir := sandboxKindSegment(kind)
		return sandboxPath(key, subdir)
	default:
		return "", fmt.Errorf("unknown storage scope '%s'", scope)
	}
}

func sandboxPath(key, subdir string) (string, error) {
	if key == "" {
		return "", errors.New("sandbox scope requires authenticated username")
	}
	if ChariotConfig.SandboxRoot != "" && filepath.IsAbs(ChariotConfig.SandboxRoot) {
		return filepath.Join(ChariotConfig.SandboxRoot, key, subdir), nil
	}
	if ChariotConfig.DataPath == "" {
		return "", errors.New("data path not configured")
	}
	sandboxBase := "sandboxes"
	if ChariotConfig.SandboxRoot != "" && ChariotConfig.SandboxRoot != "data/sandboxes" {
		sandboxBase = ChariotConfig.SandboxRoot
	}
	return filepath.Join(ChariotConfig.DataPath, sandboxBase, key, subdir), nil
}

func globalKindPath(kind StorageKind) (string, error) {
	switch kind {
	case StorageKindData:
		if ChariotConfig.DataPath == "" {
			return "", errors.New("data path not configured")
		}
		return ChariotConfig.DataPath, nil
	case StorageKindTree:
		if ChariotConfig.TreePath == "" {
			return "", errors.New("tree path not configured")
		}
		return ChariotConfig.TreePath, nil
	case StorageKindDiagram:
		if ChariotConfig.DiagramPath == "" {
			return "", errors.New("diagram path not configured")
		}
		return ChariotConfig.DiagramPath, nil
	default:
		return "", fmt.Errorf("unsupported storage kind '%s'", kind)
	}
}

func sandboxKindSegment(kind StorageKind) string {
	switch kind {
	case StorageKindData:
		return "data"
	case StorageKindTree:
		return "trees"
	case StorageKindDiagram:
		return "diagrams"
	default:
		return string(kind)
	}
}

// EnsureSandboxDirectories creates all required sandbox subdirectories for a user if they don't exist.
func EnsureSandboxDirectories(username string) error {
	if !ChariotConfig.SandboxEnabled {
		ChariotLogger.Debug("Sandbox directories not created - sandboxes disabled")
		return nil // Sandboxes disabled, nothing to do
	}
	key := SanitizeSandboxKey(username)
	if key == "" {
		ChariotLogger.Error("Invalid username for sandbox creation", zap.String("username", username))
		return errors.New("invalid username for sandbox creation")
	}
	basePath := ChariotConfig.DataPath
	if ChariotConfig.SandboxRoot != "" && filepath.IsAbs(ChariotConfig.SandboxRoot) {
		basePath = ChariotConfig.SandboxRoot
	} else if basePath == "" {
		ChariotLogger.Error("DataPath not configured")
		return errors.New("data path not configured")
	}

	ChariotLogger.Info("Ensuring sandbox directories",
		zap.String("username", username),
		zap.String("sanitizedKey", key),
		zap.String("basePath", basePath),
	)

	// Create all kind-specific directories using DataPath (same logic as storageBasePath)
	kinds := []StorageKind{StorageKindData, StorageKindTree, StorageKindDiagram}
	for _, kind := range kinds {
		path, err := sandboxPath(key, sandboxKindSegment(kind))
		if err != nil {
			ChariotLogger.Error("Failed to resolve sandbox directory",
				zap.String("kind", string(kind)),
				zap.Error(err),
			)
			return err
		}
		ChariotLogger.Debug("Creating sandbox directory",
			zap.String("path", path),
			zap.String("kind", string(kind)),
		)
		if err := os.MkdirAll(path, 0o755); err != nil {
			ChariotLogger.Error("Failed to create sandbox directory",
				zap.String("path", path),
				zap.Error(err),
			)
			return fmt.Errorf("create sandbox directory %s: %w", path, err)
		}
		ChariotLogger.Info("Created sandbox directory",
			zap.String("path", path),
			zap.String("kind", string(kind)),
		)
	}
	return nil
}
