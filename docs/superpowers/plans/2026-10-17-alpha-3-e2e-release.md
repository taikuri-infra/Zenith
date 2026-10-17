# Alpha 3 — E2E Suite and alpha.1 Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An E2E suite installs Zenith on a fresh Ubuntu server inside a CI job, drives it only through the API and the `zen` CLI, and proves all six alpha exit criteria on amd64 and arm64; then `v0.1.0` (alpha.1) is tagged from a green `main`.

**Architecture:** Six lanes with disjoint files, run in parallel on worktrees and merged by the controller: compose+installer (self-signed mode, image repo variable), API (trust the install CA), CLI installer (same `.env` keys), harness (`infra/e2e`: DinD fresh server, in-job registry, probe app, orchestration), workflow+docs (`e2e.yml`, `ci.yml`, docs), suite (`cli/e2e`, Go, build tag `e2e`). The lanes meet only through the contracts below.

**Tech Stack:** bash (installer, harness), Docker-in-Docker (`ubuntu:24.04` + Docker CE), Go 1.25 (API, CLI, suite), GitHub Actions + `act`, `registry:2`, `sslip.io`.

**Spec:** docs/superpowers/specs/2026-10-17-alpha-3-e2e-release-design.md

## Global Constraints

- Only `install.sh` is the install under test (the README one-liner); the harness never runs `zen install`.
- The suite talks only to the API and runs `zen` (locally, or on the server through `ZENITH_E2E_SERVER_EXEC`). It never runs `docker`.
- SaaS mode unchanged; `http01` installs unchanged (every new behaviour sits behind `ZENITH_TLS_MODE=selfsigned` or a new env variable with the old default).
- Verbatim names: env keys `ZENITH_TLS_MODE`, `ZENITH_TLS_CA_FILE`, `ZENITH_CERT_RESOLVER`, `ZENITH_IMAGE_REPO`; CA path inside containers `/etc/zenith/certs/zenith-ca.crt`; files `traefik/certs/zenith.crt`, `zenith.key`, `zenith-ca.crt`, `zenith-ca.key`, `traefik/dynamic/certs.yml`; registry host `zenith-e2e-registry:5000`; network `zenith-e2e`; server containers `zenith-e2e-<name>`; candidate version `<VERSION>-e2e.<sha7>`; E2E admin email `e2e@zenith.local`; probe image `zenith-e2e-registry:5000/e2e-probe:<candidate>`.
- Go: `gofmt -l` clean on touched lines; `go vet ./... && go test ./... -race -count=1` pass in `services/api` and `cli`; `cli/go.mod` stays at `go 1.25.0` (host toolchain is 1.26; never `go get …@latest`). Bash: `bash -n` and `shellcheck`-clean where shellcheck is installed.
- Commits: conventional, scope from `[api, web, cli, ci, infra, installer]`, lower-case subject, header ≤ 100 chars, trailer `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`; author `git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com'`.
- `VERSION` becomes `0.1.0` in this plan (Task 7).

## Review Focus

1. **An install.sh run in `selfsigned` mode on a box that already has a `.env`** (upgrade path): the mode must not rewrite secrets, must add the three keys only when absent, and must not break an `http01` install that re-runs the script (Task 1 test `selfsigned keys are added once and http01 stays untouched`).
2. **A server whose CA file is missing or unreadable** (an `http01` install, or a bad mount): the API must fall back to system roots with a warning, never crash or refuse all certificates (Task 2 test `TestCertPool_MissingFileFallsBackToSystem`).
3. **The arm64 runner is slower**: every wait in the suite is bounded and logs the last state; the harness kills a server that never answers `/health` within 300 s instead of hanging the job (Task 4 `wait_health` test; Task 6 `waitFor` helper test).
4. **A candidate whose `VERSION` equals the previous release's** (no bump on main): `0.1.0-alpha.4-e2e.abc1234` must still count as newer than `0.1.0-alpha.4` for `zen upgrade` (Task 4 `run.sh --plan` prints the comparison; Task 6 `TestUpgrade` asserts the server reports the candidate afterwards). If the comparison ever fails, the harness passes `--force`.
5. **The second `go test` run on a restored server** must log in with the *backed-up* admin, not the new install's generated one (Task 6 `TestBackupRestore` uses `ZENITH_E2E_EXPECT_ADMIN_EMAIL` and the password the harness carried over).

## Execution: lanes, worktrees, contracts

```bash
cd /opt/Zenith
for l in installer api cli harness ci suite; do git worktree add .worktrees/3-$l -b feat/alpha-3-$l feat/alpha-3-e2e-release; done
```

| Lane | Task | Touches |
|------|------|---------|
| installer | 1 | `docker-compose.yml`, `.env.example`, `infra/scripts/install.sh`, `infra/scripts/install_test.sh` |
| api | 2 | `services/api/**` |
| cli | 3 | `cli/internal/install/**`, `cli/internal/tlscert/**` |
| harness | 4 | `infra/e2e/**` |
| ci | 5 | `.github/workflows/e2e.yml`, `.github/workflows/ci.yml`, `docs/ci-cd-pipelines.md`, `README.md` |
| suite | 6 | `cli/e2e/**` |
| controller | 7–9 | merges, `VERSION`, `act` run, PR, release, core-audit §9 |

**Contracts (read before any task):**

- **`.env` keys written by `install.sh` in selfsigned mode:** `ZENITH_TLS_MODE=selfsigned`, `ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt`, `ZENITH_CERT_RESOLVER=` (empty). In `http01` mode nothing new is written. When `ZENITH_IMAGE_REPO` is set in the installer's environment, `.env` gets `ZENITH_IMAGE_REPO=<value>`.
- **Compose:** `image: ${ZENITH_IMAGE_REPO:-ghcr.io/taikuri-infra}/zenith-api:${ZENITH_VERSION:-latest}` (same for web); api env `ZENITH_CERT_RESOLVER: ${ZENITH_CERT_RESOLVER-le}` and `ZENITH_TLS_CA_FILE: ${ZENITH_TLS_CA_FILE:-}`; api volume `./traefik/certs:/etc/zenith/certs:ro`.
- **API:** `config.Config.TLSCAFile` from `ZENITH_TLS_CA_FILE`; `services.CertPoolWithFile(path string) (*x509.CertPool, error)` returns the system pool plus the PEM at `path` (empty path → system pool, nil error; unreadable → system pool and the error); the domain checker's prober uses it.
- **Suite env (set by the harness; names from the spec §2):** `ZENITH_E2E_ENDPOINT`, `ZENITH_E2E_DOMAIN`, `ZENITH_E2E_ADMIN_EMAIL`, `ZENITH_E2E_ADMIN_PASSWORD`, `ZENITH_E2E_CA_FILE`, `ZENITH_E2E_ZEN`, `ZENITH_E2E_INSTALL_SECONDS`, `ZENITH_E2E_SERVER_EXEC`, `ZENITH_E2E_REBOOT_CMD`, `ZENITH_E2E_COPY_FROM_SERVER`, `ZENITH_E2E_CANDIDATE_VERSION`, `ZENITH_E2E_PROBE_IMAGE`, `ZENITH_E2E_BACKUP_FILE`, `ZENITH_E2E_EXPECT_ADMIN_EMAIL`, `ZENITH_E2E_EXPECT_ADMIN_PASSWORD`, `ZENITH_E2E_ARTIFACTS` (dir for the suite's own logs).
- **Hooks:** `ZENITH_E2E_SERVER_EXEC` is a command prefix; the suite appends the command words (`<prefix> zen backup --dir /opt/zenith`). `ZENITH_E2E_REBOOT_CMD` is a full command; it returns 0 only when `/health` answers again. `ZENITH_E2E_COPY_FROM_SERVER <remote> <local>` copies a file out. All three are split on whitespace (no shell quoting inside).
- **Probe app** (`infra/e2e/probe`, Go, built by the harness, deployed by the suite as the sample app): listens on `$PORT` (default 8080); `GET /` → `ok`; `GET /write?v=<text>` → `INSERT INTO e2e(v) VALUES($1)` (creates the table if missing) using `DATABASE_URL`; `GET /read` → JSON array of all `v`; `GET /env` → JSON of the names of env vars that start with `DATABASE_URL`, `REDIS_URL`, `MYSQL_URL`. Any DB error → 500 with the error text.
- **Harness outputs** per server: `<artifacts>/<name>.env` (the suite env, one `KEY=value` per line), `<artifacts>/<name>-ca.crt`, logs under `<artifacts>/<name>/`.
- **Candidate version:** `CAND="$(tr -d '[:space:]' < VERSION)-e2e.$(git rev-parse --short=7 HEAD)"`; git tag `v$CAND` in the bare repo the harness ships; images `zenith-e2e-registry:5000/zenith-{api,web}:$CAND` and `…/e2e-probe:$CAND`.
- **Previous release for the upgrade test:** `git tag --list 'v*' --sort=-v:refname | grep -vE 'e2e|zen-|zenith-' | grep -v "^v$(tr -d '[:space:]' < VERSION)$" | head -1` (a plain `v<semver>` tag older than the candidate); `ZENITH_E2E_UPGRADE_FROM` overrides; none → `run.sh` skips `TestUpgrade` and prints why.

---

### Task 1: Compose variables and `install.sh` self-signed mode

**Lane:** installer. **Files:**
- Modify: `docker-compose.yml:24` (cert resolver), `:43` and `:104` (images), api `environment` and `volumes` blocks (`:16-50`)
- Modify: `.env.example` (document `ZENITH_IMAGE_REPO`, `ZENITH_TLS_MODE`, `ZENITH_TLS_CA_FILE`, `ZENITH_CERT_RESOLVER`)
- Modify: `infra/scripts/install.sh` (new functions after `gen_secret`; calls in the `.env` generation and the summary)
- Modify: `infra/scripts/install_test.sh` (new cases)

**Interfaces:**
- Produces: the `.env` keys and compose variables of the contracts; `install.sh` env inputs `ZENITH_TLS_MODE` (`http01`|`selfsigned`, default `http01`), `ZENITH_IMAGE_REPO`, `ADMIN_PASSWORD` (honoured when set).

- [ ] **Step 1: Failing installer tests**

Append to `infra/scripts/install_test.sh` (before the final `echo "install_test: ok"` line; keep the existing cases):

```bash
# --- self-signed mode renders certs, Traefik config and the three env keys ---
run "$t/ss" ZENITH_TLS_MODE=selfsigned
env_ss="$t/ss/.env"
grep -qE '^ZENITH_TLS_MODE=selfsigned$' "$env_ss" || fail "ZENITH_TLS_MODE missing"
grep -qE '^ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt$' "$env_ss" || fail "ZENITH_TLS_CA_FILE missing"
grep -qE '^ZENITH_CERT_RESOLVER=$' "$env_ss" || fail "ZENITH_CERT_RESOLVER must be empty in selfsigned mode"
for f in traefik/certs/zenith.crt traefik/certs/zenith.key traefik/certs/zenith-ca.crt traefik/certs/zenith-ca.key traefik/dynamic/certs.yml traefik/traefik.yml; do
  [ -s "$t/ss/$f" ] || fail "selfsigned: $f not written"
done
grep -q 'certResolver' "$t/ss/traefik/traefik.yml" && fail "selfsigned traefik.yml must not name a certResolver"
grep -qE '^certificatesResolvers:' "$t/ss/traefik/traefik.yml" && fail "selfsigned traefik.yml must not define certificatesResolvers"
openssl x509 -in "$t/ss/traefik/certs/zenith.crt" -noout -ext subjectAltName | grep -q 'DNS:\*\.203.0.113.7.sslip.io' || fail "leaf lacks the wildcard SAN"
openssl verify -CAfile "$t/ss/traefik/certs/zenith-ca.crt" "$t/ss/traefik/certs/zenith.crt" >/dev/null 2>&1 || fail "leaf does not verify against the CA"
diff <(sed -n '/^tls:/,$p' "$t/ss/traefik/dynamic/certs.yml") - <<'EOF' || fail "certs.yml content differs from the Go installer's"
tls:
  stores:
    default:
      defaultCertificate:
        certFile: /etc/traefik/certs/zenith.crt
        keyFile: /etc/traefik/certs/zenith.key
  certificates:
    - certFile: /etc/traefik/certs/zenith.crt
      keyFile: /etc/traefik/certs/zenith.key
EOF

# --- http01 (default) is untouched and image repo passes through ---
run "$t/http" ZENITH_IMAGE_REPO=zenith-e2e-registry:5000
grep -qE '^ZENITH_TLS_MODE=' "$t/http/.env" && fail "http01 must not write ZENITH_TLS_MODE"
grep -qE '^ZENITH_IMAGE_REPO=zenith-e2e-registry:5000$' "$t/http/.env" || fail "ZENITH_IMAGE_REPO not written"
[ -e "$t/http/traefik/dynamic/certs.yml" ] && fail "http01 must not write certs.yml"

# --- selfsigned keys are added once and http01 stays untouched on re-run ---
run "$t/ss" ZENITH_TLS_MODE=selfsigned
[ "$(grep -c '^ZENITH_TLS_MODE=' "$env_ss")" = 1 ] || fail "ZENITH_TLS_MODE duplicated on re-run"
cp "$t/http/.env" "$t/http/.env.before"; run "$t/http"; diff -q "$t/http/.env" "$t/http/.env.before" >/dev/null || fail "re-run changed an http01 .env"

# --- ADMIN_PASSWORD from the environment is honoured ---
run "$t/pw" ADMIN_PASSWORD=Fixed-Password-123
grep -qE '^ADMIN_PASSWORD=Fixed-Password-123$' "$t/pw/.env" || fail "ADMIN_PASSWORD override ignored"
```

The dry-run `run` helper copies `.env.example` and `VERSION` only; the self-signed branch also needs `traefik/traefik.yml` from the repo. Change `run` to also `cp -r "$repo/traefik" "$dir/traefik"` (the installer edits the copy in place).

Run: `bash infra/scripts/install_test.sh` → FAIL at the first new assertion.

- [ ] **Step 2: `install.sh` changes**

After `gen_secret`, add:

```bash
# ---- TLS mode -----------------------------------------------------------------
# http01 (default): Traefik asks Let's Encrypt over HTTP-01; the server needs a public IP.
# selfsigned: a local CA + wildcard leaf for ZENITH_DOMAIN; for LAN/offline boxes
# and for servers that cannot be reached from the internet. Same files `zen
# install --edition compose --tls-mode selfsigned` writes.
TLS_MODE="${ZENITH_TLS_MODE:-http01}"
case "$TLS_MODE" in http01|selfsigned) ;; *) err "ZENITH_TLS_MODE must be http01 or selfsigned (got '$TLS_MODE')"; exit 1;; esac

# write_selfsigned_tls DOMAIN — CA (10 y) + leaf (2 y, SAN DOMAIN and *.DOMAIN)
# under traefik/certs, the file-provider config, and a traefik.yml without an
# ACME resolver. Idempotent: existing certs are kept.
write_selfsigned_tls() {
  local domain="$1" certs="traefik/certs"
  mkdir -p "$certs" traefik/dynamic
  if [ ! -s "$certs/zenith-ca.crt" ]; then
    openssl ecparam -name prime256v1 -genkey -noout -out "$certs/zenith-ca.key"
    openssl req -x509 -new -key "$certs/zenith-ca.key" -sha256 -days 3650 -subj "/CN=Zenith Local CA" \
      -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign" -out "$certs/zenith-ca.crt"
  fi
  if [ ! -s "$certs/zenith.crt" ]; then
    openssl ecparam -name prime256v1 -genkey -noout -out "$certs/zenith.key"
    openssl req -new -key "$certs/zenith.key" -subj "/CN=$domain" -out "$certs/zenith.csr"
    printf 'subjectAltName=DNS:%s,DNS:*.%s\nextendedKeyUsage=serverAuth\nkeyUsage=digitalSignature,keyEncipherment\n' "$domain" "$domain" > "$certs/leaf.ext"
    openssl x509 -req -in "$certs/zenith.csr" -CA "$certs/zenith-ca.crt" -CAkey "$certs/zenith-ca.key" -CAcreateserial \
      -days 730 -sha256 -extfile "$certs/leaf.ext" -out "$certs/zenith.crt"
    cat "$certs/zenith-ca.crt" >> "$certs/zenith.crt"   # chain, as the Go installer writes it
    rm -f "$certs/zenith.csr" "$certs/leaf.ext" "$certs/zenith-ca.srl"
    chmod 600 "$certs"/*.key
  fi
  cat > traefik/dynamic/certs.yml <<'EOF'
tls:
  stores:
    default:
      defaultCertificate:
        certFile: /etc/traefik/certs/zenith.crt
        keyFile: /etc/traefik/certs/zenith.key
  certificates:
    - certFile: /etc/traefik/certs/zenith.crt
      keyFile: /etc/traefik/certs/zenith.key
EOF
  # Static config without any ACME resolver (Traefik then serves the default certificate).
  cat > traefik/traefik.yml <<'EOF'
# Generated by install.sh (ZENITH_TLS_MODE=selfsigned). Re-run the installer to change the TLS mode.
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"
providers:
  docker:
    exposedByDefault: false
    network: zenith_default
  file:
    directory: /etc/traefik/dynamic
    watch: true
EOF
}
```

In the `.env` generation block (after `set_var ZENITH_DOMAIN "$ZENITH_DOMAIN"`), and in the "already exists" block (guarded by `has_var`), add:

```bash
  ADMIN_PASSWORD="${ADMIN_PASSWORD:-$(gen_secret 20)}"      # replaces the unconditional generation line
  ...
  if [ -n "${ZENITH_IMAGE_REPO:-}" ]; then set_var ZENITH_IMAGE_REPO "$ZENITH_IMAGE_REPO"; fi
  if [ "$TLS_MODE" = "selfsigned" ]; then
    write_selfsigned_tls "$ZENITH_DOMAIN"
    set_var ZENITH_TLS_MODE selfsigned
    set_var ZENITH_TLS_CA_FILE /etc/zenith/certs/zenith-ca.crt
    set_var ZENITH_CERT_RESOLVER ""
  fi
```

(`set_var KEY ""` must write `KEY=` — check `set_var` handles an empty value; it does, via `printf '%s=%s'`.) In the re-run block: `if [ "$TLS_MODE" = "selfsigned" ] && ! has_var ZENITH_TLS_MODE; then … same four lines … fi`. The summary prints, in selfsigned mode: `  TLS: self-signed. Trust the CA on your machine: <install dir>/traefik/certs/zenith-ca.crt`. `write_selfsigned_tls` runs in dry-run too (it needs only openssl and the copied `traefik/` dir).

- [ ] **Step 3: Compose and `.env.example`**

`docker-compose.yml`:

```yaml
    image: ${ZENITH_IMAGE_REPO:-ghcr.io/taikuri-infra}/zenith-api:${ZENITH_VERSION:-latest}
```
```yaml
    image: ${ZENITH_IMAGE_REPO:-ghcr.io/taikuri-infra}/zenith-web:${ZENITH_VERSION:-latest}
```
api `environment`: replace `ZENITH_CERT_RESOLVER: le` with `ZENITH_CERT_RESOLVER: ${ZENITH_CERT_RESOLVER-le}` (note the single dash: an empty value stays empty) and add `ZENITH_TLS_CA_FILE: ${ZENITH_TLS_CA_FILE:-}`. api `volumes`: add `- ./traefik/certs:/etc/zenith/certs:ro` (create `traefik/certs/.gitkeep` so the bind mount exists on http01 installs). `.env.example`: a commented block for the four keys with one line each.

Run: `bash infra/scripts/install_test.sh && bash infra/scripts/compose_networks_test.sh` → `install_test: ok`; `ZENITH_DOMAIN=x docker compose config -q`; `ZENITH_DOMAIN=x ZENITH_CERT_RESOLVER= docker compose config | grep -A1 ZENITH_CERT_RESOLVER` shows an empty value; `ZENITH_IMAGE_REPO=localhost:5000 docker compose config | grep image:` shows `localhost:5000/zenith-api`.

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env.example infra/scripts/install.sh infra/scripts/install_test.sh traefik/certs/.gitkeep
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(installer): self-signed tls mode, image repo and cert resolver variables

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 2: The API trusts the install CA

**Lane:** api. **Files:**
- Modify: `services/api/internal/config/config.go` (field + `getEnv("ZENITH_TLS_CA_FILE", "")`)
- Create: `services/api/internal/services/certpool.go`, `services/api/internal/services/certpool_test.go`
- Modify: `services/api/cmd/server/main.go:1357` (prober roots)

**Interfaces:**
- Produces: `config.Config.TLSCAFile string`; `services.CertPoolWithFile(path string) (*x509.CertPool, error)`.

- [ ] **Step 1: Failing test**

```go
// services/api/internal/services/certpool_test.go
package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func selfSignedCA(t *testing.T) []byte {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Zenith Test CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestCertPool_EmptyPathIsSystemPool(t *testing.T) {
	pool, err := CertPoolWithFile("")
	if err != nil || pool == nil {
		t.Fatalf("pool=%v err=%v", pool, err)
	}
}

func TestCertPool_AddsTheFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	os.WriteFile(path, selfSignedCA(t), 0o644)
	pool, err := CertPoolWithFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sys, _ := x509.SystemCertPool()
	if sys != nil && pool.Equal(sys) {
		t.Fatal("pool must contain the CA in addition to the system roots")
	}
}

func TestCertPool_MissingFileFallsBackToSystem(t *testing.T) {
	pool, err := CertPoolWithFile(filepath.Join(t.TempDir(), "nope.crt"))
	if pool == nil || err == nil {
		t.Fatalf("want system pool and an error, got pool=%v err=%v", pool, err)
	}
}

func TestCertPool_GarbageFileFallsBackToSystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.crt")
	os.WriteFile(path, []byte("not a pem"), 0o644)
	pool, err := CertPoolWithFile(path)
	if pool == nil || err == nil {
		t.Fatalf("want system pool and an error, got pool=%v err=%v", pool, err)
	}
}
```

Run: `cd services/api && go test ./internal/services/ -run TestCertPool -count=1` → FAIL to compile.

- [ ] **Step 2: Implement**

```go
// services/api/internal/services/certpool.go
package services

import (
	"crypto/x509"
	"fmt"
	"os"
)

// CertPoolWithFile returns the system root pool plus the PEM certificates in
// path (the install's own CA on a self-signed install). An empty path means the
// system pool alone. A missing or unparsable file still returns the system pool,
// together with the error, so callers can warn and continue.
func CertPoolWithFile(path string) (*x509.CertPool, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if path == "" {
		return pool, nil
	}
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return pool, fmt.Errorf("read tls ca file %s: %w", path, err)
	}
	if !pool.AppendCertsFromPEM(pemBytes) {
		return pool, fmt.Errorf("tls ca file %s: no certificates found", path)
	}
	return pool, nil
}
```

`config.go`: field `TLSCAFile string // ZENITH_TLS_CA_FILE: PEM CA to trust in addition to the system roots (self-signed installs)` and `TLSCAFile: getEnv("ZENITH_TLS_CA_FILE", ""),` in `Load`. `main.go` where the checker is built:

```go
		roots, rerr := services.CertPoolWithFile(cfg.TLSCAFile)
		if rerr != nil {
			slog.Warn("custom domains: using system roots only", "error", rerr)
		}
		domainChecker := services.NewDomainChecker(domainRepo, net.DefaultResolver, &services.TraefikTLSProber{Roots: roots}, cfg.BaseDomain)
```

Run: `go build ./... && go vet ./... && go test ./internal/services/ ./internal/config/ -race -count=1` → PASS; then the full suite once.

- [ ] **Step 3: Commit**

```bash
git add services/api
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(api): domain checker trusts the install ca from zenith_tls_ca_file

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 3: `zen install --tls-mode selfsigned` writes the same `.env` keys

**Lane:** cli. **Files:**
- Modify: `cli/internal/install/compose.go:433-455` (`buildComposeEnv`), `cli/internal/install/compose_test.go` (new case), `cli/internal/install/traefik.go` (doc comment only: mention `install.sh` writes the same files)

**Interfaces:**
- Produces: `.env` from `zen install` contains, in selfsigned mode, `ZENITH_TLS_MODE=selfsigned`, `ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt`, `ZENITH_CERT_RESOLVER=`; in other modes none of them.

- [ ] **Step 1: Failing test** (append to `compose_test.go`, in the style of the existing `buildComposeEnv` tests):

```go
func TestBuildComposeEnv_SelfSignedWritesTLSKeys(t *testing.T) {
	cfg := &Config{Domain: "zenith.lan", AdminPassword: "pw", TLSMode: TLSSelfSigned}
	env := buildComposeEnv(cfg, "admin@zenith.lan", "999")
	for _, want := range []string{"ZENITH_TLS_MODE=selfsigned\n", "ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt\n", "ZENITH_CERT_RESOLVER=\n"} {
		if !strings.Contains(env, want) {
			t.Errorf("missing %q in:\n%s", want, env)
		}
	}
	plain := buildComposeEnv(&Config{Domain: "zenith.example.com", AdminPassword: "pw"}, "a@b.c", "999")
	for _, no := range []string{"ZENITH_TLS_MODE", "ZENITH_TLS_CA_FILE", "ZENITH_CERT_RESOLVER"} {
		if strings.Contains(plain, no) {
			t.Errorf("http01 env must not contain %s", no)
		}
	}
}
```

Run: `cd cli && go test ./internal/install/ -run TestBuildComposeEnv_SelfSigned -count=1` → FAIL.

- [ ] **Step 2: Implement** — in `buildComposeEnv`, after the `CF_DNS_API_TOKEN` block (still inside the domain `if`, since self-signed needs a domain):

```go
		if tlsMode(cfg) == TLSSelfSigned {
			b.WriteString("ZENITH_TLS_MODE=selfsigned\n")
			b.WriteString("ZENITH_TLS_CA_FILE=/etc/zenith/certs/zenith-ca.crt\n")
			b.WriteString("ZENITH_CERT_RESOLVER=\n")
		}
```

Add a sentence to the `writeSelfSignedCerts` doc comment: `install.sh (ZENITH_TLS_MODE=selfsigned) writes the same files with openssl; keep the two in step.` Run: `gofmt -l . && go vet ./... && go test ./internal/install/ ./internal/tlscert/ -race -count=1` → PASS; full suite once.

- [ ] **Step 3: Commit**

```bash
git add cli
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(cli): zen install self-signed mode writes the tls env keys the api reads

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 4: The harness — fresh server, registry, probe app, orchestration

**Lane:** harness. **Files:**
- Create: `infra/e2e/Dockerfile.server`, `infra/e2e/daemon.json`, `infra/e2e/server.sh`, `infra/e2e/run.sh`, `infra/e2e/server_test.sh`, `infra/e2e/probe/main.go`, `infra/e2e/probe/go.mod`, `infra/e2e/probe/Dockerfile`, `infra/e2e/README.md`
- Modify: `.gitignore` (add `infra/e2e/artifacts/`)

**Interfaces:**
- Consumes: the contracts (env names, registry, candidate version, previous tag); Task 1's installer env inputs.
- Produces: `server.sh {new|install|reboot|exec|cp|rm|logs|env} …`, `run.sh [--plan]`, the probe image; `<artifacts>/<name>.env` files the suite reads.

- [ ] **Step 1: Probe app**

```go
// infra/e2e/probe/main.go
// e2e-probe: the sample app the E2E suite deploys. It reads and writes one table
// through DATABASE_URL so the suite can prove data survives reboot, upgrade and restore.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ok") })
	http.HandleFunc("/env", func(w http.ResponseWriter, r *http.Request) {
		var names []string
		for _, kv := range os.Environ() {
			k := strings.SplitN(kv, "=", 2)[0]
			if strings.HasPrefix(k, "DATABASE_URL") || strings.HasPrefix(k, "REDIS_URL") || strings.HasPrefix(k, "MYSQL_URL") {
				names = append(names, k)
			}
		}
		json.NewEncoder(w).Encode(names)
	})
	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("v")
		if err := withDB(r.Context(), func(db *sql.DB) error {
			if _, err := db.ExecContext(r.Context(), `CREATE TABLE IF NOT EXISTS e2e (id serial primary key, v text not null)`); err != nil {
				return err
			}
			_, err := db.ExecContext(r.Context(), `INSERT INTO e2e(v) VALUES ($1)`, v)
			return err
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Fprint(w, "written")
	})
	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		var out []string
		if err := withDB(r.Context(), func(db *sql.DB) error {
			rows, err := db.QueryContext(r.Context(), `SELECT v FROM e2e ORDER BY id`)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var v string
				if err := rows.Scan(&v); err != nil {
					return err
				}
				out = append(out, v)
			}
			return rows.Err()
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if out == nil {
			out = []string{}
		}
		json.NewEncoder(w).Encode(out)
	})
	log.Printf("e2e-probe listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func withDB(ctx context.Context, fn func(*sql.DB) error) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return fn(db)
}
```

`infra/e2e/probe/go.mod`: `module github.com/dotechhq/zenith/infra/e2e/probe`, `go 1.25`, require `github.com/jackc/pgx/v5` at the version `services/api/go.mod` uses (`grep jackc/pgx services/api/go.mod`); run `go mod tidy` inside `infra/e2e/probe` (network is available; keep `go 1.25`). `infra/e2e/probe/Dockerfile`: multi-stage `golang:1.25-alpine` build (`CGO_ENABLED=0`) → `gcr.io/distroless/static` (or `alpine:3.20`), `EXPOSE 8080`, `ENTRYPOINT ["/probe"]`. Verify: `cd infra/e2e/probe && go vet ./... && go build -o /dev/null .`.

- [ ] **Step 2: Server image**

```dockerfile
# infra/e2e/Dockerfile.server — a "fresh Ubuntu server" for the E2E suite: Ubuntu 24.04
# with Docker CE, running dockerd as PID 1 (privileged). Nothing Zenith-specific is baked in;
# install.sh runs inside exactly as a user would run it.
FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git openssl gnupg iproute2 iptables procps \
 && install -m 0755 -d /etc/apt/keyrings \
 && curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc \
 && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable" > /etc/apt/sources.list.d/docker.list \
 && apt-get update && apt-get install -y --no-install-recommends docker-ce docker-ce-cli containerd.io docker-compose-plugin \
 && rm -rf /var/lib/apt/lists/*
COPY daemon.json /etc/docker/daemon.json
VOLUME /var/lib/docker
ENTRYPOINT ["dockerd", "--host=unix:///var/run/docker.sock"]
```

`infra/e2e/daemon.json`: `{"insecure-registries": ["zenith-e2e-registry:5000"], "storage-driver": "overlay2"}`.

- [ ] **Step 3: `server.sh`**

Write `infra/e2e/server.sh` (`#!/usr/bin/env bash`, `set -euo pipefail`) with these functions and the dispatch `case "$1" in new|install|reboot|exec|cp|rm|logs|env|ip`:

```bash
NET=zenith-e2e; REG=zenith-e2e-registry; IMG=zenith-e2e-server:local
ART="${ZENITH_E2E_ARTIFACTS:-$(cd "$(dirname "$0")" && pwd)/artifacts}"
here="$(cd "$(dirname "$0")" && pwd)"; repo="$(cd "$here/../.." && pwd)"
log() { printf '\033[36m[e2e]\033[0m %s\n' "$*" >&2; }
cname() { echo "zenith-e2e-$1"; }

ensure_network() { docker network inspect "$NET" >/dev/null 2>&1 || docker network create "$NET" >/dev/null
  # Under act the job itself is a container: join it to the network so container IPs are reachable.
  if [ -n "${ACT:-}" ] && [ -f /.dockerenv ]; then self=$(hostname); docker network connect "$NET" "$self" 2>/dev/null || true; fi; }
ensure_registry() { docker inspect "$REG" >/dev/null 2>&1 || docker run -d --name "$REG" --network "$NET" --restart unless-stopped registry:2 >/dev/null; }
ensure_image() { docker image inspect "$IMG" >/dev/null 2>&1 || docker build -q -t "$IMG" -f "$here/Dockerfile.server" "$here" >/dev/null; }
ip_of() { docker inspect -f "{{(index .NetworkSettings.Networks \"$NET\").IPAddress}}" "$(cname "$1")"; }

cmd_new() { local name="$1"; ensure_network; ensure_registry; ensure_image
  docker rm -f "$(cname "$name")" >/dev/null 2>&1 || true
  docker run -d --privileged --name "$(cname "$name")" --network "$NET" --hostname "$name" "$IMG" >/dev/null
  for _ in $(seq 1 60); do docker exec "$(cname "$name")" docker info >/dev/null 2>&1 && return 0; sleep 2; done
  log "dockerd in $(cname "$name") did not come up"; return 1; }

# cmd_install NAME candidate|release TAG — runs install.sh inside the server from a bare repo.
cmd_install() { local name="$1" mode="$2" tag="${3:-}" c; c="$(cname "$name")"
  local ip; ip="$(ip_of "$name")"; local domain="$ip.sslip.io"
  local bare; bare="$(mktemp -d)"; git clone -q --bare "$repo" "$bare/zenith.git"
  if [ "$mode" = candidate ]; then tag="v$(candidate_version)"; git -C "$repo" tag -f "$tag" HEAD >/dev/null; git -C "$bare/zenith.git" fetch -q "$repo" "refs/tags/$tag:refs/tags/$tag"; fi
  docker cp "$bare/zenith.git" "$c:/src/zenith.git"; rm -rf "$bare"
  docker cp "${ZENITH_E2E_ZEN:?}" "$c:/usr/local/bin/zen"
  local envs=(ZENITH_REPO_URL=/src/zenith.git "ZENITH_BRANCH=$tag" ZENITH_DIR=/opt/zenith "ZENITH_DOMAIN=$domain" "ZENITH_PUBLIC_IP=$ip" ZENITH_TLS_MODE=selfsigned ADMIN_EMAIL=e2e@zenith.local)
  [ "$mode" = candidate ] && envs+=("ZENITH_IMAGE_REPO=$REG:5000")
  [ -n "${ZENITH_E2E_ADMIN_PASSWORD_OVERRIDE:-}" ] && envs+=("ADMIN_PASSWORD=$ZENITH_E2E_ADMIN_PASSWORD_OVERRIDE")
  local start; start=$(date +%s)
  docker exec "${envs[@]/#/-e}" "$c" bash -c 'cd / && curl -fsSL file:///src/zenith.git/HEAD >/dev/null; bash <(git --git-dir=/src/zenith.git show "$ZENITH_BRANCH:infra/scripts/install.sh")' >"$ART/$name-install.log" 2>&1
  local secs=$(( $(date +%s) - start ))
  wait_health "$name" 300
  mkdir -p "$ART"; docker cp "$c:/opt/zenith/traefik/certs/zenith-ca.crt" "$ART/$name-ca.crt"
  local pw; pw="$(docker exec "$c" sh -c "grep '^ADMIN_PASSWORD=' /opt/zenith/.env | cut -d= -f2-")"
  cmd_env "$name" "$domain" "$pw" "$secs" > "$ART/$name.env"; log "$name installed in ${secs}s at https://$domain"; }
```

(The install command runs the `install.sh` of the chosen tag directly from the bare repo so the candidate's installer is what is tested; the `curl file://` line only fails fast if the bare repo is missing.) Also:

```bash
wait_health() { local name="$1" limit="$2" ip; ip="$(ip_of "$name")"
  for _ in $(seq 1 "$limit"); do curl -skf --max-time 3 "https://$ip.sslip.io/health" --resolve "$ip.sslip.io:443:$ip" >/dev/null 2>&1 && return 0; sleep 1; done
  log "$name: /health not answering after ${limit}s"; cmd_logs "$name"; return 1; }
cmd_reboot() { docker restart "$(cname "$1")" >/dev/null; wait_health "$1" 300; }
cmd_exec() { local name="$1"; shift; [ "${1:-}" = "--" ] && shift; docker exec -w /opt/zenith "$(cname "$name")" "$@"; }
cmd_cp() { docker cp "$(cname "$1"):$2" "$3"; }
cmd_rm() { docker rm -f "$(cname "$1")" >/dev/null 2>&1 || true; }
cmd_logs() { local name="$1" d="$ART/$name"; mkdir -p "$d"; docker logs "$(cname "$name")" > "$d/dockerd.log" 2>&1 || true
  docker exec -w /opt/zenith "$(cname "$name")" docker compose logs --no-color > "$d/compose.log" 2>&1 || true
  docker exec "$(cname "$name")" docker ps -a > "$d/docker-ps.txt" 2>&1 || true; }
# cmd_env NAME DOMAIN PASSWORD SECONDS — prints the suite env for a server.
cmd_env() { local name="$1" domain="$2" pw="$3" secs="$4"
  printf 'ZENITH_E2E_ENDPOINT=https://%s\nZENITH_E2E_DOMAIN=%s\nZENITH_E2E_ADMIN_EMAIL=e2e@zenith.local\nZENITH_E2E_ADMIN_PASSWORD=%s\n' "$domain" "$domain" "$pw"
  printf 'ZENITH_E2E_CA_FILE=%s/%s-ca.crt\nZENITH_E2E_ZEN=%s\nZENITH_E2E_INSTALL_SECONDS=%s\n' "$ART" "$name" "${ZENITH_E2E_ZEN:-zen}" "$secs"
  printf 'ZENITH_E2E_SERVER_EXEC=%s exec %s --\nZENITH_E2E_REBOOT_CMD=%s reboot %s\nZENITH_E2E_COPY_FROM_SERVER=%s cp %s\n' "$here/server.sh" "$name" "$here/server.sh" "$name" "$here/server.sh" "$name"
  printf 'ZENITH_E2E_CANDIDATE_VERSION=%s\nZENITH_E2E_PROBE_IMAGE=%s:5000/e2e-probe:%s\nZENITH_E2E_ARTIFACTS=%s\n' "$(candidate_version)" "$REG" "$(candidate_version)" "$ART"; }
candidate_version() { echo "$(tr -d '[:space:]' < "$repo/VERSION")-e2e.$(git -C "$repo" rev-parse --short=7 HEAD)"; }
```

The suite resolves `<ip>.sslip.io` through public DNS; `wait_health` uses `--resolve` so the harness does not depend on it.

- [ ] **Step 4: `run.sh`**

```bash
#!/usr/bin/env bash
# infra/e2e/run.sh — the whole E2E run: build candidate, server A (fresh candidate),
# server C (restore), server B (previous release → upgrade). Used by e2e.yml and by act.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"; repo="$(cd "$here/../.." && pwd)"; S="$here/server.sh"
export ZENITH_E2E_ARTIFACTS="${ZENITH_E2E_ARTIFACTS:-$here/artifacts}"; mkdir -p "$ZENITH_E2E_ARTIFACTS"
CAND="$(tr -d '[:space:]' < "$repo/VERSION")-e2e.$(git -C "$repo" rev-parse --short=7 HEAD)"
PREV="${ZENITH_E2E_UPGRADE_FROM:-$(git -C "$repo" tag --list 'v*' --sort=-v:refname | grep -vE 'e2e|zen-|zenith-' | grep -v "^v$(tr -d '[:space:]' < "$repo/VERSION")$" | head -1)}"
if [ "${1:-}" = "--plan" ]; then echo "candidate=$CAND previous=${PREV:-none}"; echo "A: install candidate → TestInstall TestDeploy TestReboot TestBackup"; echo "C: install candidate + restore → TestBackupRestore"; echo "B: install $PREV → TestUpgrade"; exit 0; fi
cleanup() { "$S" logs a || true; "$S" logs b || true; "$S" logs c || true; "$S" rm a; "$S" rm b; "$S" rm c; }
trap cleanup EXIT
log() { printf '\033[36m[e2e]\033[0m %s\n' "$*"; }

log "building zen and images for $CAND"
(cd "$repo/cli" && go build -o "$ZENITH_E2E_ARTIFACTS/zen" .)
export ZENITH_E2E_ZEN="$ZENITH_E2E_ARTIFACTS/zen"
"$S" new a   # also creates the network and the registry
docker build -q -t "zenith-e2e-registry:5000/zenith-api:$CAND" --build-arg "VERSION=$CAND" -f "$repo/services/api/Dockerfile" "$repo" >/dev/null
docker build -q -t "zenith-e2e-registry:5000/zenith-web:$CAND" --build-arg "VERSION=$CAND" --build-arg NEXT_PUBLIC_ZENITH_MODE=standalone -f "$repo/apps/web/Dockerfile" "$repo" >/dev/null
docker build -q -t "zenith-e2e-registry:5000/e2e-probe:$CAND" "$here/probe" >/dev/null
# push through the network name: the job host resolves the registry by connecting to its IP
REGIP=$(docker inspect -f '{{(index .NetworkSettings.Networks "zenith-e2e").IPAddress}}' zenith-e2e-registry)
for i in zenith-api zenith-web e2e-probe; do docker tag "zenith-e2e-registry:5000/$i:$CAND" "$REGIP:5000/$i:$CAND"; docker push -q "$REGIP:5000/$i:$CAND"; done

gotest() { local envfile="$1"; shift; (cd "$repo/cli" && env $(grep -v '^#' "$envfile" | xargs -d '\n') go test -tags e2e -count=1 -timeout 40m -v ./e2e/ "$@" 2>&1 | tee "$ZENITH_E2E_ARTIFACTS/$(basename "$envfile" .env)-test.log"); }

log "server A: fresh candidate install"; "$S" install a candidate
gotest "$ZENITH_E2E_ARTIFACTS/a.env" -run 'TestInstall|TestDeployWithDatabaseAndDomain|TestReboot|TestBackup$'
A_PW=$(grep '^ZENITH_E2E_ADMIN_PASSWORD=' "$ZENITH_E2E_ARTIFACTS/a.env" | cut -d= -f2-)
BACKUP=$(ls -t "$ZENITH_E2E_ARTIFACTS"/zenith-backup-*.tar.gz | head -1)

log "server C: fresh candidate install + restore of A's archive"; "$S" new c; "$S" install c candidate
docker cp "$BACKUP" "zenith-e2e-c:/opt/zenith/restore.tar.gz"
{ cat "$ZENITH_E2E_ARTIFACTS/c.env"; echo "ZENITH_E2E_BACKUP_FILE=/opt/zenith/restore.tar.gz"; echo "ZENITH_E2E_EXPECT_ADMIN_EMAIL=e2e@zenith.local"; echo "ZENITH_E2E_EXPECT_ADMIN_PASSWORD=$A_PW"; } > "$ZENITH_E2E_ARTIFACTS/c-restore.env"
gotest "$ZENITH_E2E_ARTIFACTS/c-restore.env" -run 'TestBackupRestore'
"$S" rm a; "$S" rm c

if [ -z "$PREV" ]; then log "no previous release tag found; skipping TestUpgrade"; exit 0; fi
log "server B: install $PREV then upgrade to $CAND"; "$S" new b; "$S" install b release "$PREV"
gotest "$ZENITH_E2E_ARTIFACTS/b.env" -run 'TestUpgrade'
log "E2E complete for $CAND"
```

The host pushes to the registry by IP because the job host does not resolve the Docker network name; the Docker daemon on the host must allow that insecure registry — `run.sh` checks `docker info --format '{{json .RegistryConfig.IndexConfigs}}'` and, if `$REGIP:5000` is not listed, falls back to running the pushes from a helper container on the network (`docker run --rm --network zenith-e2e -v /var/run/docker.sock:/var/run/docker.sock docker:27-cli sh -c 'docker tag … && docker push zenith-e2e-registry:5000/…'`). Implement the fallback as the default path (it works everywhere) and keep the direct push as the fast path when allowed.

- [ ] **Step 5: `server_test.sh`** (no Docker): asserts `Dockerfile.server` starts with `FROM ubuntu:24.04` and installs `docker-ce`; `daemon.json` parses with `python3 -c 'import json,sys; json.load(open(sys.argv[1]))'` and lists `zenith-e2e-registry:5000`; `bash -n server.sh run.sh`; `server.sh env demo 1.2.3.4.sslip.io pw 42` prints every `ZENITH_E2E_*` name of the contract exactly once; `run.sh --plan` prints `candidate=` and three server lines; `probe/main.go` `go vet`s. Ends with `echo "e2e_harness_test: ok"`. Add `infra/e2e/artifacts/` to `.gitignore`. `infra/e2e/README.md`: one page on what the harness does and how to run it by hand (`ZENITH_E2E_ZEN=… bash infra/e2e/run.sh`).

- [ ] **Step 6: Smoke the pieces on this host** (Docker available): `bash infra/e2e/server_test.sh`; `bash infra/e2e/server.sh new smoke && bash infra/e2e/server.sh exec smoke -- docker info --format '{{.ServerVersion}}' && bash infra/e2e/server.sh rm smoke` → prints a Docker version. Do not run the full `run.sh` here (the controller does, with all lanes merged).

- [ ] **Step 7: Commit**

```bash
git add infra/e2e .gitignore
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "feat(infra): e2e harness — fresh ubuntu server in docker, in-job registry, probe app, run script

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 5: Workflow, CI hooks, docs

**Lane:** ci. **Files:**
- Create: `.github/workflows/e2e.yml`
- Modify: `.github/workflows/ci.yml` (after the install script tests: `bash infra/e2e/server_test.sh`; after "Test zen CLI": `cd cli && go vet -tags e2e ./e2e/`)
- Modify: `docs/ci-cd-pipelines.md` (new section "E2E (`e2e.yml`)" with the job table row and the `act` command), `README.md` (Quick Start: the one-liner, what you get, `ZENITH_TLS_MODE=selfsigned` for LAN/offline, link to releases for `zen`)

- [ ] **Step 1: `e2e.yml`**

```yaml
name: E2E

# Installs Zenith on a fresh Ubuntu server (Docker-in-Docker) inside the job and
# drives it through the API and the zen CLI: install, deploy + database + custom
# domain + HTTPS, reboot, backup/restore, upgrade from the previous release.
# Runs locally with: act pull_request -W .github/workflows/e2e.yml -j e2e --container-architecture linux/amd64 --privileged

on:
  pull_request:
    paths:
      - 'services/api/**'
      - 'cli/**'
      - 'apps/web/**'
      - 'docker-compose.yml'
      - 'traefik/**'
      - 'infra/scripts/**'
      - 'infra/e2e/**'
      - 'VERSION'
      - '.github/workflows/e2e.yml'
  push:
    branches: [main]
  workflow_dispatch:

concurrency:
  group: e2e-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read

jobs:
  e2e:
    name: E2E (${{ matrix.arch }})
    runs-on: ${{ matrix.runner }}
    timeout-minutes: 60
    strategy:
      fail-fast: false
      matrix:
        include:
          - { runner: ubuntu-latest, arch: amd64 }
          - { runner: ubuntu-24.04-arm, arch: arm64 }
    steps:
      - name: Skip arm64 under act
        if: ${{ env.ACT && matrix.arch != 'amd64' }}
        run: echo "arm64 is only run on GitHub" && exit 0
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: cli/go.mod
          cache-dependency-path: cli/go.sum
      - name: Run the E2E suite on a fresh server
        run: bash infra/e2e/run.sh
      - name: Upload artifacts
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: e2e-${{ matrix.arch }}
          path: infra/e2e/artifacts/**
          if-no-files-found: ignore
```

(`act` cannot evaluate `runs-on: ubuntu-24.04-arm`; the first step exits the arm job early under act — verify with `act pull_request -W .github/workflows/e2e.yml -l`. If act refuses the arm runner label outright, add `-P ubuntu-24.04-arm=catthehacker/ubuntu:act-latest` to the documented command.)

- [ ] **Step 2: `ci.yml` and docs** as listed; `docs/ci-cd-pipelines.md` gets, in the style of §2's table: `| 1 | e2e | fresh-server install, deploy+db+domain+https, reboot, backup/restore, upgrade; amd64 + arm64 | act pull_request -W .github/workflows/e2e.yml -j e2e --privileged |` and a paragraph on the artifacts and the ~25 min duration. README Quick Start rewrite (keep it under 40 lines): one-liner, the summary it prints, the dashboard URL, `ZENITH_TLS_MODE=selfsigned` note, `zen` from the releases page, link to `docs/alpha/core-audit.md`.

- [ ] **Step 3: Validate and commit** — `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/e2e.yml')); yaml.safe_load(open('.github/workflows/ci.yml'))"`; `act pull_request -W .github/workflows/e2e.yml -l` lists the job.

```bash
git add .github/workflows/e2e.yml .github/workflows/ci.yml docs/ci-cd-pipelines.md README.md
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "ci: e2e workflow on amd64 and arm64, harness tests in ci, docs and readme

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 6: The E2E suite (`cli/e2e`)

**Lane:** suite. **Files:**
- Create: `cli/e2e/config.go`, `cli/e2e/config_test.go` (no build tag), `cli/e2e/helpers_test.go`, `cli/e2e/install_test.go`, `cli/e2e/deploy_test.go`, `cli/e2e/reboot_test.go`, `cli/e2e/backup_test.go`, `cli/e2e/upgrade_test.go` (all `//go:build e2e` except `config.go`/`config_test.go`), `cli/e2e/doc.go`

**Interfaces:**
- Consumes: the suite env contract; `cliapi` client (`LoginWithPassword`, `ListApps`, `Deploy`, `GetDeploymentStatus`, `ListAllDatabases`, `CreateDatabase`, `GetDatabaseByID`, `ServerInfo`, `DoRaw`); the probe app's HTTP surface; `zen` subcommands `login`, `status`, `deploy --image`, `backup`, `backup restore <archive>`, `upgrade --version`.
- Produces: the five tests named in the spec; `e2e.Config` and `e2e.Load()`; `e2e.SplitCmd(string) []string`.

- [ ] **Step 1: Config (untagged) with test**

```go
// cli/e2e/config.go
package e2e

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is everything the suite needs; the harness provides it as environment variables.
type Config struct {
	Endpoint, Domain, AdminEmail, AdminPassword, CAFile, Zen string
	InstallSeconds                                           int
	ServerExec, RebootCmd, CopyFromServer                     []string
	CandidateVersion, ProbeImage, Artifacts                   string
	BackupFile, ExpectAdminEmail, ExpectAdminPassword         string
}

// Load reads ZENITH_E2E_* and reports the first missing required variable.
func Load() (Config, error) {
	get := func(k string) string { return strings.TrimSpace(os.Getenv("ZENITH_E2E_" + k)) }
	c := Config{Endpoint: get("ENDPOINT"), Domain: get("DOMAIN"), AdminEmail: get("ADMIN_EMAIL"), AdminPassword: get("ADMIN_PASSWORD"),
		CAFile: get("CA_FILE"), Zen: get("ZEN"), CandidateVersion: get("CANDIDATE_VERSION"), ProbeImage: get("PROBE_IMAGE"),
		Artifacts: get("ARTIFACTS"), BackupFile: get("BACKUP_FILE"), ExpectAdminEmail: get("EXPECT_ADMIN_EMAIL"), ExpectAdminPassword: get("EXPECT_ADMIN_PASSWORD"),
		ServerExec: SplitCmd(get("SERVER_EXEC")), RebootCmd: SplitCmd(get("REBOOT_CMD")), CopyFromServer: SplitCmd(get("COPY_FROM_SERVER"))}
	for _, req := range [][2]string{{"ENDPOINT", c.Endpoint}, {"DOMAIN", c.Domain}, {"ADMIN_EMAIL", c.AdminEmail}, {"ADMIN_PASSWORD", c.AdminPassword}, {"ZEN", c.Zen}} {
		if req[1] == "" {
			return c, fmt.Errorf("ZENITH_E2E_%s is required", req[0])
		}
	}
	if s := get("INSTALL_SECONDS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			return c, fmt.Errorf("ZENITH_E2E_INSTALL_SECONDS: %w", err)
		}
		c.InstallSeconds = n
	}
	if c.Zen == "" {
		c.Zen = "zen"
	}
	return c, nil
}

// SplitCmd splits a hook on whitespace; an empty string is no hook.
func SplitCmd(s string) []string { return strings.Fields(s) }
```

`config_test.go`: sets the env with `t.Setenv`, asserts `Load` fills every field, errors name the missing variable, and `SplitCmd("docker exec -w /opt/zenith zenith-e2e-a --")` has 6 words.

- [ ] **Step 2: Helpers (`helpers_test.go`, tagged)** — a package-level `cfg` loaded in `TestMain` (fails fast on a bad config and writes `<Artifacts>/suite.log`); `httpClient()` with `RootCAs` from `cfg.CAFile` when set; `api()` → `*cliapi.Client` built with `NewClient(cfg.Endpoint, "")` whose `HTTPClient` uses that transport, logged in once with `LoginWithPassword`; `zen(args ...string) (string, error)` running `cfg.Zen` with `HOME` set to a temp dir (so `zen login` writes a throwaway config), `SSL_CERT_FILE=cfg.CAFile` and `ZEN_PASSWORD`; `serverExec(args ...string)` = `cfg.ServerExec + args` via `exec.Command`, output captured to the log; `waitFor(t, name string, timeout time.Duration, fn func() (bool, string))` polling every 5 s, failing with the last state; `getURL(t, url) (int, string)`; `probe(path) string` → `GET https://<app host>/<path>`. Every helper logs the command or URL it ran.

- [ ] **Step 3: The tests** — implement exactly the assertions of spec §2:

```go
//go:build e2e

package e2e

func TestInstall(t *testing.T) {
	if cfg.InstallSeconds <= 0 || cfg.InstallSeconds > 900 { t.Fatalf("install took %ds (limit 900)", cfg.InstallSeconds) }
	code, _ := getURL(t, cfg.Endpoint+"/health"); if code != 200 { t.Fatalf("/health %d", code) }
	code, _ = getURL(t, cfg.Endpoint+"/"); if code != 200 { t.Fatalf("dashboard %d", code) }
	info, err := api().ServerInfo(); if err != nil || info.Mode != "standalone" { t.Fatalf("version: %+v %v", info, err) }
	if cfg.CandidateVersion != "" && info.Version != cfg.CandidateVersion { t.Fatalf("version %q, want %q", info.Version, cfg.CandidateVersion) }
	code, body := postJSON(t, cfg.Endpoint+"/api/v1/auth/register", `{"email":"x@y.z","password":"longenough","name":"x"}`); if code != 403 { t.Fatalf("register %d %s", code, body) }
	out, err := zen("login", "--endpoint", cfg.Endpoint, "--email", cfg.AdminEmail); if err != nil || !strings.Contains(out, "Logged in") { t.Fatalf("zen login: %v\n%s", err, out) }
}
```

`TestDeployWithDatabaseAndDomain`: `zen deploy --image $ProbeImage` with app name `probe` (or `POST /api/v1/apps` + deploy through the client, whichever `zen deploy` uses today) → wait `running` (≤ 300 s) and the app URL (from `ListApps`) answers `ok`; `CreateDatabase` Postgres attached (`POST /api/v1/apps/<id>/databases {"name":"pg","engine":"postgresql"}`) → wait `ready` (≤ 300 s); wait until `probe("/env")` lists `DATABASE_URL` (the attach redeploys the app; ≤ 180 s); `probe("/write?v=alpha-1")` → `written`; `probe("/read")` contains `alpha-1`; env-v2 of the app lists `DATABASE_URL` with `source` `managed_service`; add domain `myapp.<Domain>` (`POST /api/v1/apps/<id>/domains`) → wait `status` `active` and `tls_ready` (≤ 180 s; log the message on each poll); `GET https://myapp.<Domain>/` → 200 and the TLS chain verifies with the CA (the http client does that). State needed by later tests (app id, app host, domain) is stored in `<Artifacts>/state.json` so `TestReboot`/`TestBackup` can run in the same process or a later one.

`TestReboot`: load state; run `cfg.RebootCmd` (≤ 10 min); wait `/health`; app URL `ok` (≤ 300 s); `probe("/read")` has `alpha-1`; database `ready`; domain `active`; `zen status` output contains the app name and `pg`.

`TestBackup`: `serverExec("zen", "backup", "--dir", "/opt/zenith")` → parse the line containing `zenith-backup-` and `.tar.gz` (take the last path-like token); `cfg.CopyFromServer + [path, <Artifacts>/<basename>]`; assert the file exists and is > 1 KB.

`TestBackupRestore` (runs only when `cfg.BackupFile != ""`, otherwise `t.Skip`): `serverExec("zen", "backup", "restore", cfg.BackupFile, "--dir", "/opt/zenith")` (check the real flag names in `cli/cmd/backup`); wait `/health` (≤ 300 s); login with `ExpectAdminEmail`/`ExpectAdminPassword`; `ListApps` has `probe`, wait `running`; database `pg` `ready`; the probe's `/read` has `alpha-1` (≤ 300 s, the app comes back after restore).

`TestUpgrade`: deploy the probe + Postgres + write `alpha-1` (reuse the deploy helper with a `before-upgrade` marker); `serverExec("zen", "upgrade", "--version", cfg.CandidateVersion, "--dir", "/opt/zenith")` (≤ 15 min; on a non-zero exit retry once with `--force` if the output says the version is not newer); wait `/health`; `ServerInfo().Version == cfg.CandidateVersion`; app URL `ok`; `probe("/read")` has `before-upgrade`; `zen status` works.

- [ ] **Step 4: Compile and unit-test** — `cd cli && go vet -tags e2e ./e2e/ && go test ./e2e/ -run TestLoad -count=1` (the untagged config test). Run the tagged tests against the live install for a smoke of the helpers only if the controller asks; the harness run is the real test.

- [ ] **Step 5: Commit**

```bash
git add cli/e2e
git -c user.name='Babak Doraniarab' -c user.email='babak.dorani@gmail.com' commit -m "test(cli): e2e suite driving a fresh install through the api and zen

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 7: Integrate, `VERSION` 0.1.0, run the suite with `act` on this host

**Lane:** controller. Merge the six lane branches into `feat/alpha-3-e2e-release` (no conflicts expected: disjoint files), set `VERSION` to `0.1.0`, run every unit suite, then `act pull_request -W .github/workflows/e2e.yml -j e2e --container-architecture linux/amd64 --privileged` (or `bash infra/e2e/run.sh` directly when act's container cannot reach the server network). Iterate with fix dispatches until the amd64 run is green; record the wall time. Commit `chore(release): version 0.1.0` and the fixes.

### Task 8: PR, GitHub matrix, merge

Push, open the PR (title `feat: alpha 3 — e2e suite on a fresh server, self-signed installs, alpha.1 release`), wait for `ci.yml` and `e2e.yml` (amd64 + arm64) to be green, squash-merge with the same subject. If arm64 exceeds 45 min, apply the spec §8 fallback (arm runs `TestInstall|TestDeploy|TestReboot`) in a fix commit and record it.

### Task 9: Release `v0.1.0` and core-audit §9

After `e2e.yml` is green on `main` for both architectures: `gh release create v0.1.0 --target main --prerelease --latest=false --title "v0.1.0 — alpha.1 (Compose edition)" --notes-file <notes>`; watch `build-public-images.yml` and `release-cli.yml`; `docker manifest inspect ghcr.io/taikuri-infra/zenith-api:0.1.0` shows amd64 + arm64; upgrade the live install (`zen upgrade --version 0.1.0`); write `docs/alpha/core-audit.md` §9 "alpha.1 — E2E" with the run links and the test → criterion table; update memory.
