#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 --profile PROFILE --output PATH [--version VERSION]" >&2
  exit 2
}

profile=""
output=""
version="dev"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --profile) profile="${2:-}"; shift 2 ;;
    --output) output="${2:-}"; shift 2 ;;
    --version) version="${2:-}"; shift 2 ;;
    *) usage ;;
  esac
done
[[ -n "$profile" && -n "$output" ]] || usage

case "$profile" in
  slim|full|data-science|training|orchestration|serving) ;;
  *) echo "unsupported Apple profile: $profile" >&2; exit 2 ;;
esac

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT
runtime_version="${version#v}"

mkdir -p "$staging/bin" "$staging/etc/kait" "$staging/requirements"
(cd "$repo_root" && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w -X main.version=${runtime_version}" -o "$staging/bin/kait" ./cmd/kait)
(cd "$repo_root" && go run ./cmd/kait contract --hardware apple --profile "$profile") > "$staging/etc/kait/identity.json"
jq -e '.schema == 3 and .runtime == "native-macos" and .hardware == "apple" and .accelerator == "mps"' "$staging/etc/kait/identity.json" >/dev/null
jq -r '.requirements[]' "$staging/etc/kait/identity.json" > "$staging/requirements/install-order.txt"
cp "$repo_root"/requirements/*.txt "$staging/requirements/"
cp "$repo_root/deploy/macos/install.sh" "$repo_root/deploy/macos/run.sh" "$staging/"
chmod 0755 "$staging/bin/kait" "$staging/install.sh" "$staging/run.sh"

mkdir -p "$(dirname "$output")"
tar -C "$staging" -czf "$output" .
echo "created $output (apple-$profile, $version)"
