# Knapsack Library Build Fix Required

## Issue

The go-chariot build is failing with undefined reference errors because the knapsack library is missing required source files.

**Error:**
```
undefined reference to `v2::ValidateConfig(v2::Config const&, std::__cxx11::basic_string<char, std::char_traits<char>, std::allocator<char> >*)'
```

## Root Cause

The `knapsack-library/CMakeLists.txt` is missing `src/v2/Config_validate.cpp` from the library sources. The file `Config_json.cpp` calls `ValidateConfig()` but the implementation file is not being linked into the library.

## Fix Required in Knapsack Repository

**File:** `knapsack/knapsack-library/CMakeLists.txt`

**Current (BROKEN) code around line 35-50:**

```cmake
# Include V2 core sources from the main project to avoid code duplication
set(PROJ_ROOT "${CMAKE_CURRENT_LIST_DIR}/..")
list(APPEND LIB_SOURCES
  "${PROJ_ROOT}/src/v2/Data.cpp"
  "${PROJ_ROOT}/src/v2/EvalCPU.cpp"
  "${PROJ_ROOT}/src/v2/BeamSearch.cpp"
  "${PROJ_ROOT}/src/v2/Preprocess.cpp"
)

# Add Metal bridge (Objective-C++) on Apple
if(APPLE AND USE_METAL)
  set(KERNELS_METAL_DIR "${CMAKE_CURRENT_LIST_DIR}/../kernels/metal")
  list(APPEND LIB_SOURCES "${KERNELS_METAL_DIR}/metal_api.mm")
  list(APPEND LIB_SOURCES "${PROJ_ROOT}/src/v2/Config.mm")
# Add CUDA sources on CUDA builds
elseif(BUILD_CUDA)
  set(KERNELS_CUDA_DIR "${CMAKE_CURRENT_LIST_DIR}/../kernels/cuda")
  list(APPEND LIB_SOURCES "${KERNELS_CUDA_DIR}/cuda_api.cu")
  list(APPEND LIB_SOURCES "${PROJ_ROOT}/src/v2/Config_json.cpp")
else()
  list(APPEND LIB_SOURCES "${PROJ_ROOT}/src/v2/Config_json.cpp")
endif()
```

**Updated (FIXED) code:**

```cmake
# Include V2 core sources from the main project to avoid code duplication
set(PROJ_ROOT "${CMAKE_CURRENT_LIST_DIR}/..")
list(APPEND LIB_SOURCES
  "${PROJ_ROOT}/src/v2/Data.cpp"
  "${PROJ_ROOT}/src/v2/EvalCPU.cpp"
  "${PROJ_ROOT}/src/v2/BeamSearch.cpp"
  "${PROJ_ROOT}/src/v2/Preprocess.cpp"
  "${PROJ_ROOT}/src/v2/Config_validate.cpp"  # <-- ADD THIS LINE
)

# Add Metal bridge (Objective-C++) on Apple
if(APPLE AND USE_METAL)
  set(KERNELS_METAL_DIR "${CMAKE_CURRENT_LIST_DIR}/../kernels/metal")
  list(APPEND LIB_SOURCES "${KERNELS_METAL_DIR}/metal_api.mm")
  list(APPEND LIB_SOURCES "${PROJ_ROOT}/src/v2/Config.mm")
# Add CUDA sources on CUDA builds
elseif(BUILD_CUDA)
  set(KERNELS_CUDA_DIR "${CMAKE_CURRENT_LIST_DIR}/../kernels/cuda")
  list(APPEND LIB_SOURCES "${KERNELS_CUDA_DIR}/cuda_api.cu")
  list(APPEND LIB_SOURCES "${PROJ_ROOT}/src/v2/Config_json.cpp")
else()
  list(APPEND LIB_SOURCES "${PROJ_ROOT}/src/v2/Config_json.cpp")
endif()
```

## Steps to Apply Fix

1. **Navigate to knapsack repository:**
   ```bash
   cd /Users/williamhouse/go/src/github.com/bhouse1273/knapsack
   ```

2. **Edit CMakeLists.txt:**
   ```bash
   # Open in your editor
   code knapsack-library/CMakeLists.txt
   
   # Find the section around line 35-40 that lists LIB_SOURCES
   # Add this line after Preprocess.cpp:
   #   "${PROJ_ROOT}/src/v2/Config_validate.cpp"
   ```

3. **Rebuild the library with platform specification:**
   ```bash
   # Rebuild Linux CPU library (AMD64)
   docker build --platform linux/amd64 \
     -f docker/Dockerfile.linux-cpu \
     --target builder \
     -t knapsack-linux-cpu-builder-amd64 .
   
   # Extract the fixed library
   CONTAINER_ID=$(docker create knapsack-linux-cpu-builder-amd64)
   docker cp "$CONTAINER_ID:/usr/local/lib/libknapsack_cpu.a" \
     knapsack-library/lib/linux-cpu/
   docker cp "$CONTAINER_ID:/usr/local/include/knapsack_cpu.h" \
     knapsack-library/lib/linux-cpu/
   docker rm "$CONTAINER_ID"
   ```

4. **Verify the fix:**
   ```bash
   # Check that ValidateConfig symbol is now present
   nm -C knapsack-library/lib/linux-cpu/libknapsack_cpu.a | grep ValidateConfig
   
   # Should output something like:
   # 0000000000001234 T v2::ValidateConfig(v2::Config const&, std::__cxx11::basic_string<char, std::char_traits<char>, std::allocator<char> >*)
   ```

5. **Retry go-chariot build:**
   ```bash
   cd /Users/williamhouse/go/src/github.com/bhouse1273/chariot-ecosystem
   ./scripts/build-azure-cross-platform.sh v0.034 go-chariot cpu
   ```

## Additional Fix: Platform Specification in build-all-platforms.sh

The knapsack build script also needs to specify `--platform linux/amd64` when building on macOS to ensure AMD64 binaries:

**File:** `knapsack/scripts/build-all-platforms.sh`

**Line ~26 (CPU build):**
```bash
# OLD:
docker build -f docker/Dockerfile.linux-cpu --target builder -t knapsack-linux-cpu-builder .

# NEW:
docker build --platform linux/amd64 -f docker/Dockerfile.linux-cpu --target builder -t knapsack-linux-cpu-builder .
```

**Line ~38 (CUDA build):**
```bash
# OLD:
docker build -f docker/Dockerfile.linux-cuda --target builder -t knapsack-linux-cuda-builder .

# NEW:
docker build --platform linux/amd64 -f docker/Dockerfile.linux-cuda --target builder -t knapsack-linux-cuda-builder .
```

This ensures that when building on macOS ARM64, Docker creates AMD64 binaries instead of ARM64 binaries.

## Summary

Two fixes needed in the knapsack repository:

1. **CMakeLists.txt** - Add `Config_validate.cpp` to library sources
2. **build-all-platforms.sh** - Add `--platform linux/amd64` to Docker build commands

After applying these fixes and rebuilding the library, the go-chariot build should succeed.

---

**Date:** November 7, 2025  
**Status:** Awaiting fix in knapsack repository
