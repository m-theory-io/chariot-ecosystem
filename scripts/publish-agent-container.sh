#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/publish-agent-container.sh --agent <agent-file> --onstart <onstart-file> [options]

Required arguments:
  --agent <path>      Path to the exported agent tree (JSON or .secure file).
  --onstart <path>    Path to the bootstrap/onStart Chariot script that wires listeners, etc.

Options:
  --profile <name>    Container profile: production (default) or debug.
  --agent-name <name> Logical agent name used in metadata and default image tags.
  --image <name>      Docker image repository/name to tag (default: chariot-agent-<agent-name-slug>).
  --tag <tag>         Docker tag to use (default: profile name).
  --platform <value>  Target platform for docker build (default: linux/amd64).
  --keep-context      Do not delete the temporary build context (for debugging).
  -h, --help          Show this help message.
EOF
}

log() {
  echo "[publish-agent] $*"
}

abort() {
  echo "[publish-agent] ERROR: $*" >&2
  exit 1
}

# Resolve repository root (script resides in scripts/)
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)
TEMPLATE_ENTRYPOINT="$SCRIPT_DIR/templates/agent-entrypoint.sh"

PROFILE="production"
AGENT_PATH=""
ONSTART_PATH=""
AGENT_NAME=""
IMAGE_REPO=""
IMAGE_TAG=""
PLATFORM="linux/amd64"
KEEP_CONTEXT=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent)
      AGENT_PATH=${2:-}
      shift 2
      ;;
    --onstart)
      ONSTART_PATH=${2:-}
      shift 2
      ;;
    --profile)
      PROFILE=${2:-}
      shift 2
      ;;
    --agent-name)
      AGENT_NAME=${2:-}
      shift 2
      ;;
    --image)
      IMAGE_REPO=${2:-}
      shift 2
      ;;
    --tag)
      IMAGE_TAG=${2:-}
      shift 2
      ;;
    --platform)
      PLATFORM=${2:-}
      shift 2
      ;;
    --keep-context)
      KEEP_CONTEXT=true
      shift 1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage
      abort "Unknown argument: $1"
      ;;
  esac
done

[[ -n "$AGENT_PATH" ]] || { usage; abort "--agent is required"; }
[[ -n "$ONSTART_PATH" ]] || { usage; abort "--onstart is required"; }
[[ -f "$AGENT_PATH" ]] || abort "Agent file not found: $AGENT_PATH"
[[ -f "$ONSTART_PATH" ]] || abort "onStart file not found: $ONSTART_PATH"
[[ -f "$TEMPLATE_ENTRYPOINT" ]] || abort "Missing entrypoint template at $TEMPLATE_ENTRYPOINT"

profile_lower=$(echo "$PROFILE" | tr '[:upper:]' '[:lower:]')
if [[ "$profile_lower" != "production" && "$profile_lower" != "debug" ]]; then
  abort "--profile must be 'production' or 'debug'"
fi
PROFILE=$profile_lower

command -v docker >/dev/null 2>&1 || abort "Docker CLI not found in PATH"

abs_path() {
  if [[ -d "$1" ]]; then
    (cd "$1" && pwd)
  else
    local dir
    dir=$(cd "$(dirname "$1")" && pwd)
    echo "$dir/$(basename "$1")"
  fi
}

AGENT_PATH=$(abs_path "$AGENT_PATH")
ONSTART_PATH=$(abs_path "$ONSTART_PATH")

if [[ -z "$AGENT_NAME" ]]; then
  AGENT_NAME=$(basename "$AGENT_PATH")
  AGENT_NAME=${AGENT_NAME%.*}
fi

agent_slug=$(echo "$AGENT_NAME" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g' | sed -E 's/^-+//;s/-+$//')
[[ -n "$agent_slug" ]] || agent_slug="agent"

default_repo="chariot-agent-${agent_slug}"
if [[ -z "$IMAGE_REPO" ]]; then
  IMAGE_REPO=$default_repo
fi

if [[ -z "$IMAGE_TAG" ]]; then
  IMAGE_TAG=$PROFILE
fi

if [[ "$IMAGE_REPO" == *:* ]]; then
  abort "--image should not include a tag; use --tag to set the tag"
fi

IMAGE_REF="${IMAGE_REPO}:${IMAGE_TAG}"
log "Building image $IMAGE_REF for profile '$PROFILE' (agent name: $AGENT_NAME)"

TMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t "chariot-agent")
CONTEXT_DIR="$TMP_DIR/context"
mkdir -p "$CONTEXT_DIR"

cleanup() {
  if [[ "$KEEP_CONTEXT" != true ]]; then
    rm -rf "$TMP_DIR"
  else
    log "Preserving build context at $TMP_DIR"
  fi
}
trap cleanup EXIT

copy_tree() {
  local src="$1"
  local dest="$2"
  mkdir -p "$dest"
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete "$src"/ "$dest"/
  else
    cp -R "$src"/. "$dest"/
  fi
}

log "Preparing build context"
copy_tree "$REPO_ROOT/services/go-chariot" "$CONTEXT_DIR/services/go-chariot"
cp "$TEMPLATE_ENTRYPOINT" "$CONTEXT_DIR/entrypoint.sh"
chmod +x "$CONTEXT_DIR/entrypoint.sh"

agent_ext="${AGENT_PATH##*.}"
if [[ "$agent_ext" == "$AGENT_PATH" ]]; then
  agent_ext="json"
fi
agent_dest_name="agent.${agent_ext}"
cp "$AGENT_PATH" "$CONTEXT_DIR/${agent_dest_name}"
onstart_dest_name="onstart.chariot"
cp "$ONSTART_PATH" "$CONTEXT_DIR/${onstart_dest_name}"

expose_line="EXPOSE 8080"
dev_rest="false"
port_line=""
if [[ "$PROFILE" == "debug" ]]; then
  expose_line="EXPOSE 8080 8090"
  dev_rest="true"
  port_line="ENV CHARIOT_PORT=8090"
fi

agent_name_env=$(printf '%s' "$AGENT_NAME" | sed 's/\\/\\\\/g; s/"/\\"/g')

cat > "$CONTEXT_DIR/Dockerfile" <<EOF
# syntax=docker/dockerfile:1.6
FROM golang:1.24-bookworm AS builder
RUN apt-get update && apt-get install -y \
    build-essential \
    g++ \
    && rm -rf /var/lib/apt/lists/*
ENV CGO_ENABLED=1
ENV CXX=g++
ENV CC=gcc
ENV GOWORK=off
WORKDIR /build
COPY services/go-chariot/ ./go-chariot/
WORKDIR /build/go-chariot
RUN if [ ! -f "knapsack-library/lib/linux-cpu/librl_support.a" ]; then \
      echo "ERROR: RL support library not found at knapsack-library/lib/linux-cpu/librl_support.a"; \
      exit 1; \
    fi
RUN go mod download
RUN go build -tags "linux,amd64,cgo" -o /tmp/chariot-server ./cmd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libstdc++6 \
    tzdata \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /tmp/chariot-server /usr/local/bin/chariot-server
COPY ${agent_dest_name} /app/${agent_dest_name}
COPY ${onstart_dest_name} /app/${onstart_dest_name}
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENV CHARIOT_AGENT_FILE=/app/${agent_dest_name}
ENV CHARIOT_ON_START_FILE=/app/${onstart_dest_name}
ENV CHARIOT_AGENT_NAME="${agent_name_env}"
ENV CHARIOT_PROFILE=${PROFILE}
ENV CHARIOT_HEADLESS=true
ENV CHARIOT_DEV_REST_ENABLED=${dev_rest}
${port_line}
${expose_line}
ENTRYPOINT ["/entrypoint.sh"]
EOF

log "Dockerfile written to $CONTEXT_DIR/Dockerfile"

log "Building image via docker"
docker build \
  --platform "$PLATFORM" \
  -t "$IMAGE_REF" \
  "$CONTEXT_DIR"

log "Image $IMAGE_REF built successfully"
if [[ "$PROFILE" == "debug" ]]; then
  log "Run example: docker run -p 8080:8080 -p 8090:8090 -e CHARIOT_PROFILE=debug $IMAGE_REF"
else
  log "Run example: docker run -p 8080:8080 $IMAGE_REF"
fi
