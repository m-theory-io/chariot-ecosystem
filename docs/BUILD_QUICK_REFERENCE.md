# Quick Reference: Building go-chariot with Knapsack

## Prerequisites

- Docker with buildx support
- Vendored knapsack libraries checked into `services/go-chariot/knapsack-library/lib/`
- Optional (only if you need to rebuild the CGO artifacts): the upstream knapsack repo as `../knapsack/` plus its `make build-all-platforms` target (see `docs/notes/KNAPSACK_INTEGRATION_COMPLETE.md`)

## Build Commands

### Linux AMD64 CPU (Cost-Effective)
```bash
# Build Docker image
./scripts/build-azure-cross-platform.sh v0.034 go-chariot cpu

# Outputs:
# - go-chariot:v0.034-cpu
# - go-chariot:latest-cpu

# Run locally
docker run -p 8080:8080 go-chariot:v0.034-cpu
```

**Use Case:** Azure Standard VMs, development environments
**Cost:** ~$50-100/month
**Performance:** Baseline

### Linux ARM64 CUDA (High Performance)
```bash
# Build Docker image
./scripts/build-azure-cross-platform.sh v0.034 go-chariot cuda

# Outputs:
# - go-chariot:v0.034-cuda
# - go-chariot:latest-cuda

# Run with GPU
docker run --gpus all -p 8080:8080 go-chariot:v0.034-cuda
```

**Use Case:** NVIDIA Jetson, Azure NC-series VMs
**Cost:** ~$1,000+/month (Azure GPU) or $500-1,000 (Jetson hardware)
**Performance:** 10-50x faster

### macOS Metal (Local Development)
```bash
# Build locally (not in Docker)
cd services/go-chariot
CGO_ENABLED=1 go build -tags cgo -o go-chariot ./cmd

# Run
./go-chariot
```

**Use Case:** Local macOS development with GPU acceleration
**Performance:** 15-30x faster than CPU

## Build All Services
```bash
# Build all services with CPU go-chariot
./scripts/build-azure-cross-platform.sh v0.034 all cpu

# Build all services with CUDA go-chariot
./scripts/build-azure-cross-platform.sh v0.034 all cuda
```

## Platform Selection Logic

Go automatically selects the correct implementation:

| Target | Build Tags | Selected File | Library |
|--------|-----------|---------------|---------|
| Azure Standard VM | `linux,amd64,cgo` | `knapsack_cgo_linux_amd64.go` | `libknapsack_cpu.a` |
| Jetson/NC-series | `linux,arm64,cuda,cgo` | `knapsack_cgo_linux_arm64_cuda.go` | `libknapsack_cuda.a` |
| macOS Local | `darwin,cgo` | `knapsack_cgo_darwin_metal.go` | `libknapsack_metal.a` |

## Deployment to Azure

### 1. Tag and Push
```bash
# CPU variant
docker tag go-chariot:v0.034-cpu mtheorycontainerregistry.azurecr.io/go-chariot:v0.034-cpu
docker push mtheorycontainerregistry.azurecr.io/go-chariot:v0.034-cpu

# CUDA variant  
docker tag go-chariot:v0.034-cuda mtheorycontainerregistry.azurecr.io/go-chariot:v0.034-cuda
docker push mtheorycontainerregistry.azurecr.io/go-chariot:v0.034-cuda
```

### 2. Update Deployment
```bash
# Update docker-compose or k8s manifests to use platform-specific tag
# For CPU: image: mtheorycontainerregistry.azurecr.io/go-chariot:v0.034-cpu
# For GPU: image: mtheorycontainerregistry.azurecr.io/go-chariot:v0.034-cuda
```

## Troubleshooting

### Vendored knapsack libraries missing
The build script now expects the CGO artifacts inside this repository under `services/go-chariot/knapsack-library/lib/*`. If you see `Vendored CPU library not found` (or the CUDA equivalent), confirm the files exist:

```bash
ls -lh services/go-chariot/knapsack-library/lib/linux-cpu/
ls -lh services/go-chariot/knapsack-library/lib/linux-cuda/
ls -lh services/go-chariot/knapsack-library/lib/macos-cpu/
ls -lh services/go-chariot/knapsack-library/lib/macos-metal/
```

Each folder should contain the platform-specific archive (`libknapsack_cpu.a`, `libknapsack_cuda.a`, `libknapsack_macos_cpu.a`, or `libknapsack_metal.a`), the matching header (for example `knapsack_cpu.h`), plus the RL helper artifacts (`librl_support.a`, `rl_api.h`, and on macOS the `.dylib`). If any of those files are missing, copy them from the canonical knapsack repo or rebuild them as described below.

### Rebuild knapsack libraries
When you modify the solver or refresh the binaries, rebuild them from the knapsack repo (see `services/go-chariot/knapsack-library/lib/README.md` and `docs/notes/KNAPSACK_INTEGRATION_COMPLETE.md`):

```bash
cd ../knapsack
make build-all-platforms
```

After the build completes, copy the resulting archives and headers into `services/go-chariot/knapsack-library/lib/<platform>/` so the go-chariot build can vendor them.

### Build takes too long
- Pre-built libraries eliminate the need to compile knapsack in Docker
- If still slow, check Docker build cache: `docker buildx prune`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AZURE_REGISTRY` | `mtheorycontainerregistry` | Azure Container Registry name |
| `CGO_ENABLED` | `1` | Enable CGO for knapsack linking |

## Files Reference

### Platform Implementations
- `services/go-chariot/chariot/knapsack_cgo_linux_amd64.go` - AMD64 CPU
- `services/go-chariot/chariot/knapsack_cgo_linux_arm64_cuda.go` - ARM64 CUDA  
- `services/go-chariot/chariot/knapsack_cgo_darwin_metal.go` - macOS Metal
- `services/go-chariot/chariot/knapsack_stub.go` - Unsupported platforms

### Vendored CGO Libraries
- `services/go-chariot/knapsack-library/lib/linux-cpu/libknapsack_cpu.a`
- `services/go-chariot/knapsack-library/lib/linux-cuda/libknapsack_cuda.a`
- `services/go-chariot/knapsack-library/lib/macos-cpu/libknapsack_macos_cpu.a`
- `services/go-chariot/knapsack-library/lib/macos-metal/libknapsack_metal.a`
	- Each folder also provides the matching header (`knapsack_*.h`) plus RL helpers (`librl_support.a`, `rl_api.h`, and `.dylib` on macOS)

### Dockerfiles
- `infrastructure/docker/go-chariot/Dockerfile.cpu` - CPU build
- `infrastructure/docker/go-chariot/Dockerfile.cuda` - CUDA build

### Build Script
- `scripts/build-azure-cross-platform.sh` - Main build orchestrator

## Performance Expectations

| Platform | Relative Speed | Monthly Cost | Use Case |
|----------|----------------|--------------|----------|
| **CPU** | 1x | $50-100 | Dev, small loads |
| **Metal** | 15-30x | Local only | macOS dev |
| **CUDA** | 10-50x | $1,000+ | Production, large problems |

---

**Updated:** November 7, 2025
**Documentation:** See `docs/KNAPSACK_INTEGRATION_COMPLETE.md` for full details
