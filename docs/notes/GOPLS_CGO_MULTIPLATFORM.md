# Gopls Multi-Platform CGO Analysis

## The Issue

When working with platform-specific CGO files (e.g., `rl_cgo_darwin_cpu.go`, `rl_cgo_linux_cpu.go`, `knapsack_cgo_darwin_cpu.go`), gopls analyzes code for **multiple platforms simultaneously**. This results in errors like:

```
go list failed to return CompiledGoFiles [linux,amd64]
undefined: C.CString [linux,amd64]
rlInit_impl redeclared in this block [linux,amd64]
```

These errors appear because gopls is trying to analyze darwin-specific files as if they were being compiled for Linux, which correctly fails since they have `//go:build darwin && arm64 && cgo` constraints.

## Why This Happens

This is a **known limitation** of gopls (see [golang/go#38990](https://github.com/golang/go/issues/38990)). Gopls's cross-platform analysis cannot be fully disabled, even with:
- `GOOS`/`GOARCH` environment variables
- Build flags and tags
- Build environment settings
- gopls.toml configuration

## The Good News

**These errors do NOT affect your actual builds!**

✅ Local builds work: `CGO_ENABLED=1 go build ./cmd`  
✅ Tests pass: `CGO_ENABLED=1 go test ./tests`  
✅ Docker images build successfully  
✅ VM deployments work  

The `[linux,amd64]` tag in the error message indicates these are **informational** - gopls is telling you these darwin files won't compile on Linux, which is correct and expected.

## Solutions

### Option 1: Hide Red Squiggles (Current Setting)

`.vscode/settings.json`:
```json
{
  "problems.decorations.enabled": false
}
```

This hides red squiggles in the editor while keeping the Problems panel functional.

### Option 2: Filter Problems Panel

1. Open Problems panel: `Cmd+Shift+M`
2. Type in filter box: `!linux`
3. This excludes all `[linux,amd64]` errors from view

### Option 3: Accept the Noise

Understand that `[linux,amd64]` errors are expected for darwin-specific files and mentally filter them out.

## Verification

To verify your code actually builds:

```bash
cd services/go-chariot
CGO_ENABLED=1 go build -o /tmp/go-chariot ./cmd
echo "Exit code: $?"  # Should be 0
```

## Related Files

Platform-specific CGO files in this project:
- `chariot/rl_cgo_darwin_cpu.go` - macOS CPU (LinUCB + ONNX)
- `chariot/rl_cgo_darwin_metal.go` - macOS Metal GPU
- `chariot/rl_cgo_linux_cpu.go` - Linux AMD64 CPU
- `chariot/rl_cgo_linux_cuda.go` - Linux ARM64 CUDA GPU
- `chariot/knapsack_cgo_darwin_cpu.go` - macOS knapsack solver
- `chariot/knapsack_cgo_linux_amd64.go` - Linux knapsack solver
- `chariot/rl_stub.go` - Fallback when CGO disabled

Each file has appropriate `//go:build` constraints to ensure it only compiles on its target platform.

## Summary

The gopls errors are **cosmetic** and do not indicate actual problems. Your code is correctly structured with platform-specific build tags, which gopls's multi-platform analysis correctly identifies as incompatible with other platforms.
