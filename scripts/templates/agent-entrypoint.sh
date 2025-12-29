#!/usr/bin/env sh
set -euo pipefail

profile=$(printf '%s' "${CHARIOT_PROFILE:-production}" | tr '[:upper:]' '[:lower:]')

if [ "$profile" = "debug" ]; then
  export CHARIOT_DEV_REST_ENABLED="${CHARIOT_DEV_REST_ENABLED:-true}"
  : "${CHARIOT_PORT:=8090}"
else
  export CHARIOT_DEV_REST_ENABLED="${CHARIOT_DEV_REST_ENABLED:-false}"
fi

if [ -z "${CHARIOT_HEADLESS:-}" ]; then
  export CHARIOT_HEADLESS=true
fi

exec /usr/local/bin/chariot-server "$@"
