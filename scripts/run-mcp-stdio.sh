#!/usr/bin/env bash
set -euo pipefail

# Wrapper to launch go-chariot as an MCP server over stdio.
# Usage: ./scripts/run-mcp-stdio.sh [--no-build]
# By default, builds the go-chariot cmd binary and runs it with MCP stdio env.

NO_BUILD="false"
if [[ ${1:-} == "--no-build" ]]; then
  NO_BUILD="true"
fi

# Resolve repo root relative to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SERVICE_DIR="${REPO_ROOT}/services/go-chariot"

# Ensure we run from the service dir for consistent paths
cd "${SERVICE_DIR}"

BUILD_DIR="./build"
BIN_PATH="${BUILD_DIR}/go-chariot-mcp"

# Build the cmd binary unless suppressed
if [[ "${NO_BUILD}" != "true" ]]; then
  echo "[MCP] Building go-chariot cmd binary..." >&2
  mkdir -p "${BUILD_DIR}"
  go build -o "${BIN_PATH}" ./cmd
fi

# Default env (can be overridden by caller)
export CHARIOT_MCP_ENABLED="${CHARIOT_MCP_ENABLED:-true}"
export CHARIOT_MCP_TRANSPORT="${CHARIOT_MCP_TRANSPORT:-stdio}"
# Optional: disable REST explicitly (stdio mode short-circuits before REST anyway)
export CHARIOT_DEV_REST_ENABLED="${CHARIOT_DEV_REST_ENABLED:-false}"

# Run the MCP server (stdio). Process exits when the MCP client disconnects.
echo "[MCP] Launching go-chariot (stdio transport)." >&2
echo "      CHARIOT_MCP_ENABLED=${CHARIOT_MCP_ENABLED} CHARIOT_MCP_TRANSPORT=${CHARIOT_MCP_TRANSPORT}" >&2
exec "${BIN_PATH}"
