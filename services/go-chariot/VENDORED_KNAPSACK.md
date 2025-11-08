# Vendored Knapsack Libraries

## Overview

The go-chariot service now uses vendored knapsack libraries to simplify builds and eliminate external dependencies. Pre-built knapsack libraries and headers are stored in the `knapsack-library/lib/` directory.

## Directory Structure

```
services/go-chariot/knapsack-library/lib/
├── linux-cpu/           # AMD64 CPU-only (Linux)
│   ├── libknapsack_cpu.a
│   └── knapsack_cpu.h
├── linux-cuda/          # ARM64 CUDA GPU (Linux)
│   ├── libknapsack_cuda.a
│   └── knapsack_cuda.h
└── macos-metal/         # ARM64 Metal GPU (macOS)
    ├── libknapsack_metal.a
    └── knapsack_metal.h
```

## Platform Assumptions

- **linux-cpu**: Targets AMD64 architecture (Intel/AMD x86_64)
- **linux-cuda**: Targets ARM64 architecture (NVIDIA Jetson, ARM servers with CUDA)
- **macos-metal**: Targets ARM64 Apple Silicon (M1/M2/M3)

## CGO Integration

Each platform has its own CGO implementation file that references the vendored libraries:

### Linux AMD64 CPU
- File: `chariot/knapsack_cgo_linux_amd64.go`
- Build tags: `//go:build linux && amd64 && cgo`
- Library: `knapsack-library/lib/linux-cpu/libknapsack_cpu.a`
- Header: `knapsack_cpu.h`
- CGO Flags:
  ```go
  #cgo CFLAGS: -I${SRCDIR}/../knapsack-library/lib/linux-cpu
  #cgo LDFLAGS: ${SRCDIR}/../knapsack-library/lib/linux-cpu/libknapsack_cpu.a -lstdc++ -lm
  ```

### Linux ARM64 CUDA
- File: `chariot/knapsack_cgo_linux_arm64_cuda.go`
- Build tags: `//go:build linux && arm64 && cuda && cgo`
- Library: `knapsack-library/lib/linux-cuda/libknapsack_cuda.a`
- Header: `knapsack_cuda.h`
- CGO Flags:
  ```go
  #cgo CFLAGS: -I${SRCDIR}/../knapsack-library/lib/linux-cuda
  #cgo LDFLAGS: ${SRCDIR}/../knapsack-library/lib/linux-cuda/libknapsack_cuda.a -lstdc++ -lm -lcudart
  ```

### macOS ARM64 Metal
- File: `chariot/knapsack_cgo_darwin_metal.go`
- Build tags: `//go:build darwin && cgo`
- Library: `knapsack-library/lib/macos-metal/libknapsack_metal.a`
- Header: `knapsack_metal.h`
- CGO Flags:
  ```go
  #cgo CFLAGS: -I${SRCDIR}/../knapsack-library/lib/macos-metal
  #cgo LDFLAGS: ${SRCDIR}/../knapsack-library/lib/macos-metal/libknapsack_metal.a -framework Metal -framework Foundation -lstdc++ -lm
  ```

## Building

### Docker Builds

The Dockerfiles have been simplified to use vendored libraries:

**CPU (AMD64)**:
```bash
docker buildx build \
    --platform linux/amd64 \
    -f infrastructure/docker/go-chariot/Dockerfile.cpu \
    -t go-chariot:latest-cpu \
    .
```

**CUDA (ARM64)**:
```bash
docker buildx build \
    --platform linux/arm64 \
    -f infrastructure/docker/go-chariot/Dockerfile.cuda \
    -t go-chariot:latest-cuda \
    .
```

### Local Builds

**macOS (Metal)**:
```bash
cd services/go-chariot
CGO_ENABLED=1 go build -tags cgo -o bin/go-chariot ./cmd
```

**Linux (CPU)**:
```bash
cd services/go-chariot
CGO_ENABLED=1 go build -tags "linux,amd64,cgo" -o bin/go-chariot ./cmd
```

**Linux (CUDA)**:
```bash
cd services/go-chariot
CGO_ENABLED=1 go build -tags "linux,arm64,cuda,cgo" -o bin/go-chariot ./cmd
```

## Build Script

The `scripts/build-azure-cross-platform.sh` script automatically validates that vendored libraries exist before building:

```bash
# Build CPU image
./scripts/build-azure-cross-platform.sh latest go-chariot cpu

# Build CUDA image
./scripts/build-azure-cross-platform.sh latest go-chariot cuda
```

## Updating Vendored Libraries

When the knapsack library is updated, copy the new libraries to the knapsack-library directory:

```bash
# From knapsack repository root
cp knapsack-library/lib/linux-cpu/libknapsack_cpu.a \
   ../chariot-ecosystem/services/go-chariot/knapsack-library/lib/linux-cpu/

cp knapsack-library/lib/linux-cpu/knapsack_cpu.h \
   ../chariot-ecosystem/services/go-chariot/knapsack-library/lib/linux-cpu/

# Repeat for linux-cuda and macos-metal
```

## Benefits

1. **No External Dependencies**: Docker builds don't need access to the knapsack repository
2. **Simplified Dockerfiles**: Single-stage builds without build contexts
3. **Consistent Builds**: Always uses the same library version that was vendored
4. **Faster Builds**: No need to build knapsack during Docker image creation
5. **Clear Dependencies**: Library files are version-controlled with go-chariot
6. **No Go Vendor Conflicts**: Using `knapsack-library/` instead of `vendor/` avoids conflicts with `go mod vendor`

## Troubleshooting

### Missing Library Error
```
Error: Vendored CPU library not found
Expected: services/go-chariot/knapsack-library/lib/linux-cpu/libknapsack_cpu.a
```

**Solution**: Copy the pre-built library from the knapsack repository to the knapsack-library directory.

### Linker Errors
If you see undefined symbol errors, ensure:
1. The library was built for the correct platform (AMD64 vs ARM64)
2. The library includes all required symbols (check knapsack CMakeLists.txt)
3. The correct build tags are used (e.g., `cuda` tag for CUDA builds)

### Header Not Found
```
fatal error: knapsack_cpu.h: No such file or directory
```

**Solution**: Ensure the header file is in the same directory as the `.a` file and matches the platform-specific name.
