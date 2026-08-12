#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
  echo "Kait Apple GPU workers require native macOS on arm64" >&2
  exit 2
fi
if [[ -z "${BUILDKITE_AGENT_TOKEN:-}" && -z "${BUILDKITE_AGENT_TOKEN_FILE:-}" && "${KAIT_RUN_MODE:-agent}" == "agent" ]]; then
  echo "Set BUILDKITE_AGENT_TOKEN or BUILDKITE_AGENT_TOKEN_FILE" >&2
  exit 2
fi

export KAIT_HARDWARE=apple
export PATH="/opt/kait/venv/bin:/usr/local/bin:/opt/homebrew/bin:${PATH}"
exec /usr/local/bin/kait "$@"
