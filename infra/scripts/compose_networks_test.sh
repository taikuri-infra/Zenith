#!/usr/bin/env bash
# Checks the compose file keeps the control plane off the app network.
set -euo pipefail
repo="$(cd "$(dirname "$0")/../.." && pwd)"
fail() { echo "FAIL: $*" >&2; exit 1; }
cfg="$(cd "$repo" && JWT_SECRET=x ZENITH_DOMAIN=example.com docker compose -f docker-compose.yml --profile tls config --format json)"
nets() { echo "$cfg" | python3 -c "import sys,json;s=json.load(sys.stdin)['services']['$1'];print(' '.join(sorted((s.get('networks') or {'default':None}).keys())))"; }
[ "$(nets postgres)" = "internal" ] || fail "postgres networks: $(nets postgres)"
[ "$(nets storage)" = "internal" ] || fail "storage networks: $(nets storage)"
[ "$(nets api)" = "edge internal" ] || fail "api networks: $(nets api)"
[ "$(nets web)" = "edge" ] || fail "web networks: $(nets web)"
[ "$(nets traefik)" = "default edge" ] || fail "traefik networks: $(nets traefik)"
name() { echo "$cfg" | python3 -c "import sys,json;print(json.load(sys.stdin)['networks']['$1']['name'])"; }
[ "$(name default)" = "zenith_default" ] || fail "default network name"
[ "$(name internal)" = "zenith_internal" ] || fail "internal network name"
[ "$(name edge)" = "zenith_edge" ] || fail "edge network name"
echo "$cfg" | grep -q 'traefik.docker.network=zenith_edge' || echo "$cfg" | python3 -c "import sys,json;l=json.load(sys.stdin)['services']['api']['labels'];assert l.get('traefik.docker.network')=='zenith_edge'" || fail "api router network label"
echo "compose_networks_test: ok"
