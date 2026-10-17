#!/usr/bin/env bash
# infra/e2e/daemon-mirror.sh FILE MIRROR_URL — add MIRROR_URL as a registry mirror to the
# dockerd config FILE (a one-object JSON document). Idempotent: a mirror already present is
# not added twice. Used by entrypoint.sh when ZENITH_E2E_REGISTRY_MIRROR is set.
set -euo pipefail
file="${1:?usage: daemon-mirror.sh FILE MIRROR_URL}"; mirror="${2:?usage: daemon-mirror.sh FILE MIRROR_URL}"
case "$mirror" in
  https://*|http://*) ;;
  *) echo "daemon-mirror: mirror must be an http(s) URL: $mirror" >&2; exit 2;;
esac
[ -s "$file" ] || printf '{}\n' > "$file"
if grep -qF "\"$mirror\"" "$file"; then exit 0; fi
if grep -q '"registry-mirrors"' "$file"; then
  # Append to the existing list: "registry-mirrors": [ ... ] -> [ ..., "mirror"]
  sed -i "s|\(\"registry-mirrors\"[[:space:]]*:[[:space:]]*\[[^]]*\)\]|\1, \"$mirror\"]|" "$file"
  sed -i 's|\[, |[|' "$file"
elif grep -qE '^[[:space:]]*\{[[:space:]]*\}[[:space:]]*$' "$file"; then
  printf '{"registry-mirrors": ["%s"]}\n' "$mirror" > "$file"
else
  # Insert a new key right after the opening brace.
  sed -i "0,/{/s|{|{\"registry-mirrors\": [\"$mirror\"], |" "$file"
fi
