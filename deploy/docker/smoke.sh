#!/usr/bin/env bash
set -euo pipefail

export KAIT_CONTAINER_COMMAND=smoke
exec "$(dirname "${BASH_SOURCE[0]}")/run.sh"
