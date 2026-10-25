#!/usr/bin/env bash
set -euo pipefail

transport=${1:-http}
profile=${2:-production}
case "$transport" in
  http) bootstrap=bootstrap-http-only.ch ;;
  nsq) bootstrap=bootstrap.ch; : "${CHARIOT_NSQ_ADDR:?Set CHARIOT_NSQ_ADDR to a reachable NSQD TCP address}" ;;
  *) echo "usage: bash examples/listeners/run-example.sh http|nsq production|debug" >&2; exit 2 ;;
esac
case "$profile" in
  production) rest=false ;;
  debug) rest=true ;;
  *) echo "profile must be production or debug" >&2; exit 2 ;;
esac

example_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd "$example_dir/../.." && pwd)
run_dir=$(mktemp -d "${TMPDIR:-/tmp}/chariot-listeners.XXXXXX")
cp "$example_dir"/*.ch "$example_dir"/*.json "$example_dir/bootstrap-count.txt" "$run_dir/"
mkdir -p "$run_dir/listeners"
cp "$example_dir/devtest.ch" "$run_dir/listeners/"

export CHARIOT_DATA_PATH="$run_dir"
export CHARIOT_TREE_PATH="$run_dir"
export CHARIOT_DIAGRAM_PATH="$run_dir/diagrams"
export CHARIOT_SANDBOX_ROOT="$run_dir/sandboxes"
export CHARIOT_LISTENER_SCRIPTS_DIR="$run_dir/listeners"
export CHARIOT_LISTENERS_FILE=listeners.json
export CHARIOT_FUNCTION_LIB=stlib.json
export CHARIOT_BOOTSTRAP="$bootstrap"
export CHARIOT_SECRET_PROVIDER=file
export CHARIOT_SECRET_FILE_PATH="$run_dir/secrets.local.json"
export CHARIOT_HEADLESS=true
export CHARIOT_DEV_REST_ENABLED="$rest"
export CHARIOT_PORT=8090
export CHARIOT_SSL=false
export CHARIOT_MCP_ENABLED=false
export CHARIOT_MCP_EXECUTE_ENABLED=false
export CHARIOT_SHUTDOWN_TIMEOUT=30

echo "Data/counter directory: $run_dir (retained after exit for inspection)"
echo "HTTP: http://127.0.0.1:18080/health and http://127.0.0.1:18081/echo"
echo "Dev REST enabled: $rest; port: $CHARIOT_PORT"
cd "$repo_root/services/go-chariot"
go build -o "$run_dir/chariot-server" ./cmd
exec "$run_dir/chariot-server"