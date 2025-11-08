# Quick Reference: Building go-chariot with Knapsack

## Prerequisites

- Docker with buildx support
- knapsack repo as sibling directory: `../knapsack/`
- Pre-built libraries in `../knapsack/knapsack-library/lib/`

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

### "knapsack repo not found"
```bash
export KNAPSACK_REPO=/path/to/knapsack
./scripts/build-azure-cross-platform.sh v0.034 go-chariot cpu
```

### "Pre-built library not found"
```bash
# Verify libraries exist
ls -lh ../knapsack/knapsack-library/lib/linux-cpu/
ls -lh ../knapsack/knapsack-library/lib/linux-cuda/

# Rebuild if needed
cd ../knapsack && make build-all-platforms
```

### Build takes too long
- Pre-built libraries eliminate the need to compile knapsack in Docker
- If still slow, check Docker build cache: `docker buildx prune`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KNAPSACK_REPO` | `../knapsack` | Path to knapsack repository |
| `AZURE_REGISTRY` | `mtheorycontainerregistry` | Azure Container Registry name |
| `CGO_ENABLED` | `1` | Enable CGO for knapsack linking |

## Files Reference

### Platform Implementations
- `services/go-chariot/chariot/knapsack_cgo_linux_amd64.go` - AMD64 CPU
- `services/go-chariot/chariot/knapsack_cgo_linux_arm64_cuda.go` - ARM64 CUDA  
- `services/go-chariot/chariot/knapsack_cgo_darwin_metal.go` - macOS Metal
- `services/go-chariot/chariot/knapsack_stub.go` - Unsupported platforms

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
