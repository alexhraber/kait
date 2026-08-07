#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${KAITE_CONTAINER_COMMAND:-}" && "${KAITE_RUN_MODE:-agent}" == "agent" && -z "${BUILDKITE_AGENT_TOKEN:-}" && -z "${BUILDKITE_AGENT_TOKEN_FILE:-}" ]]; then
  echo "Set BUILDKITE_AGENT_TOKEN or BUILDKITE_AGENT_TOKEN_FILE" >&2
  exit 2
fi
hardware="${KAITE_HARDWARE:-cpu}"
o11y="${KAITE_O11Y:-none}"
image="${KAITE_IMAGE:-ghcr.io/alexhraber/kaite:${hardware}}"

args=(
  --rm
  --init
  --name "${KAITE_CONTAINER_NAME:-kaite-agent}"
  --publish "${KAITE_METRICS_PORT:-9090}:9090"
  --env "KAITE_HARDWARE=${hardware}"
  --env "KAITE_O11Y=${o11y}"
  --env "KAITE_METRICS_ADDR=${KAITE_METRICS_ADDR:-0.0.0.0:9090}"
  --env "OTEL_SERVICE_NAME=${OTEL_SERVICE_NAME:-kaite}"
)

forward_env() {
  local name="$1"
  if [[ -n "${!name:-}" ]]; then
    args+=(--env "${name}")
  fi
}

for name in \
  BUILDKITE_AGENT_TOKEN \
  BUILDKITE_AGENT_TOKEN_FILE \
  BUILDKITE_AGENT_TAGS \
  BUILDKITE_AGENT_NAME \
  BUILDKITE_AGENT_CONFIG \
  BUILDKITE_AGENT_ENDPOINT \
  BUILDKITE_AGENT_QUEUE \
  BUILDKITE_AGENT_PRIORITY \
  BUILDKITE_AGENT_ACQUIRE_JOB \
  BUILDKITE_AGENT_DISCONNECT_AFTER_JOB \
  BUILDKITE_AGENT_DISCONNECT_AFTER_IDLE_TIMEOUT \
  BUILDKITE_AGENT_REFLECT_EXIT_STATUS \
  BUILDKITE_AGENT_SHELL \
  BUILDKITE_AGENT_HOOKS_PATH \
  BUILDKITE_WRITE_JOB_LOGS_TO_STDOUT \
  BUILDKITE_KUBERNETES_EXEC \
  KAITE_RUN_MODE \
  KAITE_COMMAND \
  DD_AGENT_HOST \
  DD_DOGSTATSD_PORT \
  KAITE_DD_AGENT_HOST \
  KAITE_DD_DOGSTATSD_PORT \
  OTEL_EXPORTER_OTLP_ENDPOINT \
  OTEL_EXPORTER_OTLP_HEADERS \
  OTEL_EXPORTER_OTLP_PROTOCOL \
  OTEL_RESOURCE_ATTRIBUTES; do
  forward_env "${name}"
done

if [[ -n "${BUILDKITE_AGENT_TOKEN_FILE:-}" ]]; then
  args+=(--volume "${BUILDKITE_AGENT_TOKEN_FILE}:${BUILDKITE_AGENT_TOKEN_FILE}:ro")
fi

command=()
if [[ -n "${KAITE_CONTAINER_COMMAND:-}" ]]; then
  command+=("${KAITE_CONTAINER_COMMAND}")
fi

case "${hardware}" in
  nvidia) args+=(--gpus "${KAITE_GPU_DEVICES:-all}") ;;
  amd) args+=(--device /dev/kfd --device /dev/dri --group-add video) ;;
  intel) args+=(--device /dev/dri --group-add video) ;;
  cpu|apple) ;;
  *) echo "unsupported KAITE_HARDWARE=${hardware}" >&2; exit 2 ;;
esac

exec docker run "${args[@]}" "${image}" "${command[@]}"
