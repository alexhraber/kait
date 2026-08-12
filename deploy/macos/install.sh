#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" || "$(uname -m)" != "arm64" ]]; then
  echo "Kait Apple GPU workers require native macOS on arm64; do not install this bundle in a Linux VM or container." >&2
  exit 2
fi

bundle_root=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
python_bin="${KAIT_PYTHON_BIN:-$(command -v python3 || true)}"
if [[ -z "$python_bin" ]]; then
  echo "python3 is required; install a supported Python 3 through Homebrew or the organization image policy" >&2
  exit 2
fi

if [[ "$(id -u)" -eq 0 ]]; then
  sudo_cmd=()
else
  sudo_cmd=(sudo)
fi

install_root="${KAIT_INSTALL_ROOT:-/opt/kait}"
identity_root="/Library/Application Support/Kait"
"${sudo_cmd[@]}" mkdir -p "$install_root/requirements" "$identity_root" /usr/local/bin
"${sudo_cmd[@]}" cp "$bundle_root/etc/kait/identity.json" "$identity_root/identity.json"
"${sudo_cmd[@]}" cp "$bundle_root/requirements/"*.txt "$install_root/requirements/"
"${sudo_cmd[@]}" install -m 0755 "$bundle_root/bin/kait" /usr/local/bin/kait

"${sudo_cmd[@]}" "$python_bin" -m venv "$install_root/venv"
"${sudo_cmd[@]}" "$install_root/venv/bin/pip" install --upgrade pip
while IFS= read -r manifest; do
  [[ -n "$manifest" ]] || continue
  "${sudo_cmd[@]}" "$install_root/venv/bin/pip" install -r "$install_root/requirements/$manifest"
done < "$bundle_root/requirements/install-order.txt"

profile="$($python_bin -c 'import json,sys; print(json.load(open(sys.argv[1]))["profile"])' "$bundle_root/etc/kait/identity.json")"
echo "Installed Kait Apple MPS profile $profile"
echo "Install Buildkite's native macOS agent separately with: brew install buildkite/buildkite/buildkite-agent"
