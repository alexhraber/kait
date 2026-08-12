#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${KAIT_CONTAINER_COMMAND:-}" && "${KAIT_RUN_MODE:-agent}" == "agent" && -z "${BUILDKITE_AGENT_TOKEN:-}" && -z "${BUILDKITE_AGENT_TOKEN_FILE:-}" ]]; then
  echo "Set BUILDKITE_AGENT_TOKEN or BUILDKITE_AGENT_TOKEN_FILE" >&2
  exit 2
fi
hardware="${KAIT_HARDWARE:-cpu}"
variant="${KAIT_VARIANT:-slim}"
profile="${KAIT_PROFILE:-${variant}}"
o11y="${KAIT_O11Y:-none}"
case "${profile}" in
  slim|full|data-science|training|orchestration|serving) ;;
  *) echo "unsupported KAIT_PROFILE=${profile}" >&2; exit 2 ;;
esac
case "${profile}" in
  slim|data-science) variant="slim" ;;
  full|training|orchestration|serving) variant="full" ;;
esac
image="${KAIT_IMAGE:-ghcr.io/alexhraber/kait:${hardware}-${profile}}"

args=(
  --rm
  --init
  --name "${KAIT_CONTAINER_NAME:-kait-agent}"
  --publish "${KAIT_METRICS_PORT:-9090}:9090"
  --env "KAIT_HARDWARE=${hardware}"
  --env "KAIT_VARIANT=${variant}"
  --env "KAIT_PROFILE=${profile}"
  --env "KAIT_O11Y=${o11y}"
  --env "KAIT_METRICS_ADDR=${KAIT_METRICS_ADDR:-0.0.0.0:9090}"
  --env "OTEL_SERVICE_NAME=${OTEL_SERVICE_NAME:-kait}"
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
  KAIT_RUN_MODE \
  KAIT_COMMAND \
  KAIT_PROFILE \
  KAIT_CAPABILITIES \
  DD_AGENT_HOST \
  DD_DOGSTATSD_PORT \
  KAIT_DD_AGENT_HOST \
  KAIT_DD_DOGSTATSD_PORT \
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
if [[ -n "${KAIT_CONTAINER_COMMAND:-}" ]]; then
  command+=("${KAIT_CONTAINER_COMMAND}")
fi

case "${hardware}" in
  nvidia) args+=(--gpus "${KAIT_GPU_DEVICES:-all}") ;;
  amd) args+=(--device /dev/kfd --device /dev/dri --group-add video) ;;
  intel) args+=(--device /dev/dri --group-add video) ;;
  cpu) ;;
  apple)
    echo "Apple hardware is inactive; use the multi-architecture Linux CPU image with KAIT_HARDWARE=cpu" >&2
    exit 2
    ;;
  *) echo "unsupported KAIT_HARDWARE=${hardware}" >&2; exit 2 ;;
esac

exec docker run "${args[@]}" "${image}" "${command[@]}"
