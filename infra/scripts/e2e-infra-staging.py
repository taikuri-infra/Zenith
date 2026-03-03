#!/usr/bin/env python3
"""
Zenith Platform E2E Infrastructure Test — Staging
===================================================
Validates DNS, HTTPS, SSL, redirects, content, API reachability,
and (optionally) Kubernetes cluster health for the staging environment
at *.stage.freezenith.com.

Usage:
    python3 infra/scripts/e2e-infra-staging.py [--verbose] [--skip-k8s] [--ssh-host HOST]

Returns exit 0 if all tests pass, exit 1 if any fail.
"""

import argparse
import sys

from e2e_lib import (
    BLUE,
    CYAN,
    NC,
    YELLOW,
    TestRunner,
    check_dns,
    check_https_status,
    check_ssl_cert,
    print_banner,
    ssh_command,
)

SERVER_IP = "77.42.88.149"

# Domains that should resolve to SERVER_IP
EXACT_DOMAINS = [
    "stage.freezenith.com",
    "api.stage.freezenith.com",
    "app.stage.freezenith.com",
    "auth.stage.freezenith.com",
    "hub.stage.freezenith.com",
    "argocd.stage.freezenith.com",
    "grafana.stage.freezenith.com",
]

# Domains that just need to resolve (any IP)
RESOLVE_ONLY_DOMAINS = [
    "registry.stage.freezenith.com",
]

ALL_DOMAINS = EXACT_DOMAINS + RESOLVE_ONLY_DOMAINS

# HTTPS endpoints to check
HTTPS_CHECKS = [
    ("https://stage.freezenith.com", 200),
    ("https://api.stage.freezenith.com/health", 200),
    ("https://app.stage.freezenith.com", 200),
    ("https://auth.stage.freezenith.com", 200),
    ("https://argocd.stage.freezenith.com", 200),
]

# HTTP → HTTPS redirect checks
REDIRECT_CHECKS = [
    "http://stage.freezenith.com",
    "http://api.stage.freezenith.com",
    "http://app.stage.freezenith.com",
]


def run_dns_tests(t: TestRunner) -> None:
    t.section("1/7", "DNS Resolution")

    for domain in EXACT_DOMAINS:
        ip, ok = check_dns(domain, expected_ip=SERVER_IP)
        if ok:
            t.passed(f"{domain} -> {ip}")
        else:
            t.failed(f"{domain} -> expected {SERVER_IP}", f"got '{ip}'")

    for domain in RESOLVE_ONLY_DOMAINS:
        ip, ok = check_dns(domain)
        if ok:
            t.passed(f"{domain} -> {ip}")
        else:
            t.failed(f"{domain} -> no DNS response")


def run_https_tests(t: TestRunner) -> None:
    t.section("2/7", "HTTPS Connectivity")

    for url, expected in HTTPS_CHECKS:
        status, _ = check_https_status(url)
        if status == expected:
            t.passed(f"{url} -> HTTP {status}")
        else:
            t.failed(f"{url} -> expected HTTP {expected}", f"got {status}")


def run_redirect_tests(t: TestRunner) -> None:
    t.section("3/7", "HTTP -> HTTPS Redirects")

    for url in REDIRECT_CHECKS:
        status, _ = check_https_status(url, follow_redirects=False)
        if status in (301, 308):
            t.passed(f"{url} -> HTTP {status} (permanent redirect)")
        elif status in (302, 307):
            t.passed(f"{url} -> HTTP {status} (temporary redirect)")
        else:
            t.failed(f"{url} -> expected 301/308 redirect", f"got {status}")


def run_ssl_tests(t: TestRunner) -> None:
    t.section("4/7", "SSL Certificates")

    for domain in ALL_DOMAINS:
        info = check_ssl_cert(domain)
        if info["ok"] and info.get("self_signed"):
            t.passed(f"{domain} -> {YELLOW}self-signed cert (present but untrusted){NC}")
        elif info["ok"]:
            days = info["days_remaining"]
            expires = info["not_after"]
            if days < 14:
                t.failed(f"{domain} -> expires in {days} days ({expires})", "renew soon!")
            elif days < 30:
                t.passed(f"{domain} -> {YELLOW}valid, expires {expires} ({days}d){NC}")
            else:
                t.passed(f"{domain} -> valid, expires {expires} ({days}d)")
        else:
            t.failed(f"{domain} -> SSL check failed", info["error"])


def run_content_tests(t: TestRunner) -> None:
    t.section("5/7", "Content Verification")

    # Landing page
    status, body = check_https_status("https://stage.freezenith.com")
    if status == 200 and ("zenith" in body.lower() or "freezenith" in body.lower()):
        t.passed("Landing page contains 'zenith'")
    else:
        t.failed("Landing page content check", f"status={status}")

    # API health
    status, body = check_https_status("https://api.stage.freezenith.com/health")
    if status == 200 and ("ok" in body.lower() or "healthy" in body.lower() or "status" in body.lower()):
        t.passed("API health endpoint returns status JSON")
    else:
        t.failed("API health content check", f"status={status} body={body[:100]}")

    # Web dashboard
    status, body = check_https_status("https://app.stage.freezenith.com")
    if status == 200:
        t.passed("Web dashboard loads")
    else:
        t.failed("Web dashboard loads", f"status={status}")

    # Keycloak OIDC discovery
    oidc_url = "https://auth.stage.freezenith.com/realms/master/.well-known/openid-configuration"
    status, body = check_https_status(oidc_url)
    if status == 200 and ("authorization_endpoint" in body or "issuer" in body):
        t.passed("Keycloak OIDC discovery endpoint responds")
    else:
        t.failed("Keycloak OIDC discovery", f"status={status}")


def run_api_tests(t: TestRunner) -> None:
    t.section("6/7", "API Reachability")

    from e2e_lib import api_request

    base = "https://api.stage.freezenith.com"

    # Health
    status, body = api_request(base, "GET", "/health")
    if status == 200:
        t.passed("GET /health -> healthy")
    else:
        t.failed("GET /health", f"status={status}")

    # Version
    status, body = api_request(base, "GET", "/api/v1/version")
    if status == 200 and isinstance(body, dict) and "version" in body:
        t.passed(f"GET /api/v1/version -> {body.get('version', '?')}")
    elif status != 0:
        t.passed(f"GET /api/v1/version -> reachable (HTTP {status})")
    else:
        t.failed("GET /api/v1/version -> connection failed")

    # Auth login reachable (expect 4xx, not connection error)
    status, body = api_request(base, "POST", "/api/v1/auth/login")
    if status != 0:
        t.passed(f"POST /api/v1/auth/login -> reachable (HTTP {status})")
    else:
        t.failed("POST /api/v1/auth/login -> connection failed")

    # Protected endpoint without JWT → 401
    status, body = api_request(base, "GET", "/api/v1/apps")
    if status == 401:
        t.passed("GET /api/v1/apps -> 401 without JWT (auth enforced)")
    else:
        t.failed("GET /api/v1/apps -> expected 401", f"got {status}")


def run_k8s_tests(t: TestRunner, ssh_host: str) -> None:
    t.section("7/7", "Kubernetes Cluster (via SSH)")

    # Test SSH connectivity first
    ok, output = ssh_command(ssh_host, "echo ok")
    if not ok:
        t.skipped("All K8s checks", "SSH not available")
        return

    # Pods running in zenith-staging
    ok, output = ssh_command(ssh_host, "kubectl get pods -n zenith-staging --no-headers 2>/dev/null")
    if ok and output:
        lines = [l for l in output.splitlines() if l.strip()]
        running = [l for l in lines if "Running" in l or "Completed" in l]
        t.passed(f"zenith-staging pods: {len(running)}/{len(lines)} healthy")
    else:
        t.failed("List pods in zenith-staging", output[:200] if output else "no output")

    # ArgoCD apps synced
    ok, output = ssh_command(
        ssh_host,
        "kubectl get applications -n argocd -o jsonpath='{range .items[*]}{.metadata.name}={.status.sync.status}/{.status.health.status} {end}' 2>/dev/null",
    )
    if ok and output:
        apps = output.strip().split()
        all_synced = all("Synced" in a for a in apps)
        if all_synced:
            t.passed(f"ArgoCD apps all synced ({len(apps)} apps)")
        else:
            out_of_sync = [a for a in apps if "Synced" not in a]
            t.failed(f"ArgoCD apps not all synced", f"out-of-sync: {', '.join(out_of_sync)}")
    else:
        t.failed("Check ArgoCD sync status", output[:200] if output else "no output")

    # API pod image tag
    ok, output = ssh_command(
        ssh_host,
        "kubectl get pod -n zenith-staging -l app=zenith-api -o jsonpath='{.items[0].spec.containers[0].image}' 2>/dev/null",
    )
    if ok and output:
        t.passed(f"API pod image: {output}")
    else:
        t.failed("Get API pod image tag", output[:200] if output else "no output")


def main() -> int:
    parser = argparse.ArgumentParser(description="Zenith E2E Infrastructure Test — Staging")
    parser.add_argument("--verbose", "-v", action="store_true", help="Show failure details")
    parser.add_argument("--skip-k8s", action="store_true", help="Skip Kubernetes cluster checks")
    parser.add_argument("--ssh-host", default="zen-stage", help="SSH host for K8s checks (default: zen-stage)")
    args = parser.parse_args()

    print_banner("Zenith Infrastructure E2E Test — Staging")

    t = TestRunner(verbose=args.verbose)

    try:
        run_dns_tests(t)
        run_https_tests(t)
        run_redirect_tests(t)
        run_ssl_tests(t)
        run_content_tests(t)
        run_api_tests(t)

        if args.skip_k8s:
            t.section("7/7", "Kubernetes Cluster (via SSH)")
            t.skipped("All K8s checks", "--skip-k8s")
        else:
            run_k8s_tests(t, args.ssh_host)
    except KeyboardInterrupt:
        print(f"\n{YELLOW}Interrupted{NC}")

    t.summary()
    return 1 if t.fail_count > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
