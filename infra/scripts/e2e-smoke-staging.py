#!/usr/bin/env python3
"""
Zenith Platform E2E Smoke Test — Staging
=========================================
Exercises the full user lifecycle against the staging API via port-forward:
  register → create app → deploy → hit limits → IDOR check → upgrade → pro features → cleanup

Usage:
    python3 infra/scripts/e2e-smoke-staging.py [--verbose] [--no-cleanup] [--base-url URL]

Prerequisites:
    kubectl port-forward svc/zenith-api 18080:8080 -n zenith-staging

Returns exit 0 if all tests pass, exit 1 if any fail.
"""

import argparse
import json
import os
import random
import string
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from typing import Any, Optional

# ---------------------------------------------------------------------------
# Colours (ANSI)
# ---------------------------------------------------------------------------
RED = "\033[0;31m"
GREEN = "\033[0;32m"
YELLOW = "\033[1;33m"
BLUE = "\033[0;34m"
CYAN = "\033[0;36m"
BOLD = "\033[1m"
NC = "\033[0m"

# ---------------------------------------------------------------------------
# Test harness
# ---------------------------------------------------------------------------
@dataclass
class TestRunner:
    verbose: bool = False
    pass_count: int = 0
    fail_count: int = 0
    total: int = 0
    failures: list = field(default_factory=list)

    def passed(self, label: str) -> None:
        self.pass_count += 1
        self.total += 1
        print(f"  {GREEN}PASS{NC} {label}")

    def failed(self, label: str, detail: str = "") -> None:
        self.fail_count += 1
        self.total += 1
        self.failures.append(label)
        print(f"  {RED}FAIL{NC} {label}")
        if self.verbose and detail:
            print(f"       {YELLOW}Detail: {detail}{NC}")

    def section(self, num: str, title: str) -> None:
        print(f"\n{BLUE}[{num}]{NC} {BOLD}{title}{NC}")

    def summary(self) -> None:
        print()
        print("=" * 50)
        if self.fail_count == 0:
            print(f"  {GREEN}ALL {self.total} TESTS PASSED{NC}")
        else:
            print(f"  {RED}{self.fail_count} of {self.total} TESTS FAILED{NC}")
            print(f"  {GREEN}{self.pass_count} passed{NC}, {RED}{self.fail_count} failed{NC}")
            print()
            for f in self.failures:
                print(f"    {RED}✗{NC} {f}")
        print("=" * 50)
        print()


# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------
def api_request(
    base: str,
    method: str,
    path: str,
    token: str = "",
    body: Optional[dict] = None,
    expected: Optional[int] = None,
) -> tuple[int, Any]:
    """Make an HTTP request and return (status_code, parsed_json_or_text)."""
    url = f"{base}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")

    try:
        resp = urllib.request.urlopen(req, timeout=15)
        status = resp.status
        raw = resp.read().decode()
    except urllib.error.HTTPError as e:
        status = e.code
        raw = e.read().decode() if e.fp else ""
    except Exception as e:
        return 0, str(e)

    try:
        parsed = json.loads(raw) if raw else {}
    except json.JSONDecodeError:
        parsed = raw

    return status, parsed


def rand_id(n: int = 6) -> str:
    return "".join(random.choices(string.ascii_lowercase + string.digits, k=n))


# ---------------------------------------------------------------------------
# Port-forward helper
# ---------------------------------------------------------------------------
PF_PROC: Optional[subprocess.Popen] = None


def start_port_forward(namespace: str, svc: str, local_port: int, remote_port: int) -> bool:
    global PF_PROC
    try:
        PF_PROC = subprocess.Popen(
            ["kubectl", "port-forward", f"svc/{svc}", f"{local_port}:{remote_port}", "-n", namespace],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        time.sleep(2)  # give it a moment to bind
        return PF_PROC.poll() is None
    except FileNotFoundError:
        return False


def stop_port_forward() -> None:
    global PF_PROC
    if PF_PROC and PF_PROC.poll() is None:
        PF_PROC.terminate()
        PF_PROC.wait(timeout=5)
        PF_PROC = None


# ---------------------------------------------------------------------------
# Main test flow
# ---------------------------------------------------------------------------
def run_tests(base_url: str, t: TestRunner, no_cleanup: bool) -> None:
    uid = rand_id()

    # Stash IDs for cleanup
    user1_token = ""
    user2_token = ""
    app1_id = ""
    app2_id = ""
    db1_id = ""
    bucket_id = ""

    # ===================================================================
    # A. Setup — 1/7
    # ===================================================================
    t.section("1/7", "Setup & Health Check")

    status, body = api_request(base_url, "GET", "/health")
    if status == 200:
        t.passed("Health check → 200")
    else:
        t.failed("Health check → 200", f"got {status}")
        print(f"\n  {RED}API unreachable at {base_url}/health — aborting.{NC}\n")
        return

    status, body = api_request(base_url, "GET", "/api/v1/version")
    if status == 200 and isinstance(body, dict):
        ver = body.get("version", "?")
        t.passed(f"Version endpoint → {ver}")
    else:
        t.failed("Version endpoint", f"status={status}")

    # ===================================================================
    # B. Auth & Baseline — 2/7
    # ===================================================================
    t.section("2/7", "Auth & Free-Plan Baseline")

    email1 = f"smoke-{uid}-1@test.zenith.dev"
    email2 = f"smoke-{uid}-2@test.zenith.dev"
    password = f"Sm0ke-{uid}-Pass!"

    # Register user 1
    status, body = api_request(base_url, "POST", "/api/v1/auth/register", body={
        "email": email1, "password": password, "name": f"Smoke {uid}"
    })
    if status == 201 and "access_token" in body:
        user1_token = body["access_token"]
        t.passed(f"Register user1 ({email1})")
    else:
        t.failed(f"Register user1", f"status={status} body={body}")
        return  # can't continue without auth

    # Verify free plan
    status, body = api_request(base_url, "GET", "/api/v1/plan", token=user1_token)
    if status == 200 and body.get("tier") == "free":
        limits = body.get("limits", {})
        t.passed(f"Free plan assigned (max_apps={limits.get('max_apps')})")
    else:
        t.failed("Free plan check", f"status={status} tier={body.get('tier')}")

    # Verify free limits
    if status == 200:
        limits = body.get("limits", {})
        checks = [
            ("max_apps", 1),
            ("max_databases", 1),
            ("custom_domain", False),
            ("backups_enabled", False),
        ]
        all_ok = True
        for key, expected_val in checks:
            if limits.get(key) != expected_val:
                all_ok = False
                t.failed(f"Free limit {key}={expected_val}", f"got {limits.get(key)}")
        if all_ok:
            t.passed("Free tier limits correct")

    # Billing status
    status, body = api_request(base_url, "GET", "/api/v1/billing", token=user1_token)
    if status == 200 and body.get("tier") == "free":
        t.passed("Billing status → free tier")
    else:
        t.failed("Billing status", f"status={status}")

    # ===================================================================
    # C. Free Tier Resources — 3/7
    # ===================================================================
    t.section("3/7", "Free Tier — Create Resources")

    # Create app 1
    status, body = api_request(base_url, "POST", "/api/v1/apps", token=user1_token, body={
        "name": f"smoke-app-{uid}", "repo_url": "https://github.com/test/repo"
    })
    if status == 201 and body.get("id"):
        app1_id = body["id"]
        t.passed(f"Create app1 → {body.get('name')} (id={app1_id[:8]})")
    else:
        t.failed("Create app1", f"status={status} body={body}")

    # Create database 1
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/databases",
                                   token=user1_token, body={"name": f"smoke-db-{uid}", "engine": "postgres"})
        if status == 201 and body.get("id"):
            db1_id = body["id"]
            t.passed(f"Create db1 → {body.get('name')} (engine={body.get('engine')})")
        else:
            t.failed("Create db1", f"status={status} body={body}")

    # Register a release (pre-built image)
    release_id = ""
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/releases",
                                   token=user1_token, body={
                                       "image": "registry.stage.freezenith.com/zenith-stage/smoke-hello:latest",
                                       "git_sha": "abc1234",
                                       "branch": "main",
                                       "message": "smoke test release",
                                   })
        if status == 201 and body.get("id"):
            release_id = body["id"]
            t.passed(f"Create release → {release_id[:8]}")
        else:
            t.failed("Create release", f"status={status} body={body}")

    # Deploy the release
    deployment_id = ""
    if app1_id and release_id:
        status, body = api_request(base_url, "POST",
                                   f"/api/v1/apps/{app1_id}/releases/{release_id}/deploy",
                                   token=user1_token)
        if status == 202 and body.get("deployment_id"):
            deployment_id = body["deployment_id"]
            t.passed(f"Deploy release → deployment={deployment_id[:8]}")
        else:
            t.failed("Deploy release", f"status={status} body={body}")

    # List deployments
    if app1_id:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/deployments",
                                   token=user1_token)
        if status == 200 and body.get("total", 0) >= 1:
            t.passed(f"List deployments → {body['total']} found")
        else:
            t.failed("List deployments", f"status={status} total={body.get('total')}")

    # ===================================================================
    # D. Free Tier Limits — 4/7
    # ===================================================================
    t.section("4/7", "Free Tier — Enforce Limits")

    # 2nd app → should be 403
    if app1_id:
        status, body = api_request(base_url, "POST", "/api/v1/apps", token=user1_token, body={
            "name": f"smoke-app2-{uid}", "repo_url": "https://github.com/test/repo2"
        })
        if status == 403:
            t.passed("2nd app → 403 (plan limit enforced)")
        else:
            t.failed("2nd app should be 403", f"got {status}")

    # Custom domain → should be 403 (free tier)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/domains",
                                   token=user1_token, body={"domain": "custom.example.com"})
        if status == 403:
            t.passed("Custom domain → 403 (free tier)")
        else:
            t.failed("Custom domain should be 403", f"got {status}")

    # 2nd database → should be 403
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/databases",
                                   token=user1_token, body={"name": f"smoke-db2-{uid}", "engine": "postgres"})
        if status == 403:
            t.passed("2nd database → 403 (plan limit enforced)")
        else:
            t.failed("2nd database should be 403", f"got {status}")

    # 2nd storage bucket → should be 403
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/storage",
                                   token=user1_token, body={"name": f"smoke-bucket2-{uid}"})
        if status == 403:
            t.passed("2nd bucket → 403 (plan limit enforced)")
        else:
            t.failed("2nd bucket should be 403", f"got {status}")

    # Backup → should be 403 (free tier: backups_enabled=false)
    if app1_id and db1_id:
        status, body = api_request(base_url, "POST",
                                   f"/api/v1/apps/{app1_id}/databases/{db1_id}/backups",
                                   token=user1_token)
        if status == 403:
            t.passed("Backup → 403 (free tier)")
        else:
            # Backups may not be gated yet — treat as a warning
            t.failed("Backup should be 403 on free tier", f"got {status}")

    # ===================================================================
    # E. Security / IDOR — 5/7
    # ===================================================================
    t.section("5/7", "Security — IDOR & Auth")

    # Register user 2
    status, body = api_request(base_url, "POST", "/api/v1/auth/register", body={
        "email": email2, "password": password, "name": f"Smoke2 {uid}"
    })
    if status == 201 and "access_token" in body:
        user2_token = body["access_token"]
        t.passed(f"Register user2 ({email2})")
    else:
        t.failed("Register user2", f"status={status}")

    # User2 tries to access user1's app → should fail
    if app1_id and user2_token:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}",
                                   token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 → app1 = {status}")
        else:
            t.failed("IDOR: user2 should not access app1", f"got {status}")

    # User2 tries to delete user1's app
    if app1_id and user2_token:
        status, body = api_request(base_url, "DELETE", f"/api/v1/apps/{app1_id}",
                                   token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 delete app1 = {status}")
        else:
            t.failed("IDOR: user2 should not delete app1", f"got {status}")

    # User2 tries to access user1's databases
    if app1_id and user2_token:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/databases",
                                   token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 → app1 databases = {status}")
        else:
            t.failed("IDOR: user2 should not list app1 databases", f"got {status}")

    # Invalid JWT → 401
    status, body = api_request(base_url, "GET", "/api/v1/apps",
                               token="invalid.jwt.token")
    if status == 401:
        t.passed("Invalid JWT → 401")
    else:
        t.failed("Invalid JWT should be 401", f"got {status}")

    # No JWT → 401
    status, body = api_request(base_url, "GET", "/api/v1/apps")
    if status == 401:
        t.passed("No JWT → 401")
    else:
        t.failed("No JWT should be 401", f"got {status}")

    # ===================================================================
    # F. Pro Upgrade — 6/7
    # ===================================================================
    t.section("6/7", "Pro Upgrade & Pro Features")

    # Upgrade to pro
    status, body = api_request(base_url, "POST", "/api/v1/plan/upgrade",
                               token=user1_token, body={"tier": "pro"})
    if status == 200 and body.get("tier") == "pro":
        t.passed("Upgrade to pro")
    else:
        t.failed("Upgrade to pro", f"status={status} body={body}")

    # Verify pro limits
    status, body = api_request(base_url, "GET", "/api/v1/plan", token=user1_token)
    if status == 200 and body.get("tier") == "pro":
        limits = body.get("limits", {})
        if limits.get("max_apps") == 5 and limits.get("custom_domain") is True:
            t.passed(f"Pro limits correct (max_apps=5, custom_domain=true)")
        else:
            t.failed("Pro limits", f"limits={limits}")
    else:
        t.failed("Pro plan check", f"status={status}")

    # Now create 2nd app (should succeed on pro)
    status, body = api_request(base_url, "POST", "/api/v1/apps", token=user1_token, body={
        "name": f"smoke-app2-{uid}", "repo_url": "https://github.com/test/repo2"
    })
    if status == 201 and body.get("id"):
        app2_id = body["id"]
        t.passed(f"Create app2 on pro → {body.get('name')}")
    else:
        t.failed("Create app2 on pro", f"status={status} body={body}")

    # Storage bucket (pro allows buckets)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/storage",
                                   token=user1_token, body={"name": f"smoke-bucket-{uid}"})
        if status == 201 and body.get("id"):
            bucket_id = body["id"]
            t.passed(f"Create S3 bucket → {body.get('name')}")
        else:
            t.failed("Create S3 bucket", f"status={status} body={body}")

    # Custom domain (pro allows)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/domains",
                                   token=user1_token, body={"domain": f"smoke-{uid}.example.com"})
        if status == 201:
            t.passed(f"Add custom domain → {body.get('domain')}")
        else:
            t.failed("Add custom domain on pro", f"status={status} body={body}")

    # Backup (pro allows)
    if app1_id and db1_id:
        status, body = api_request(base_url, "POST",
                                   f"/api/v1/apps/{app1_id}/databases/{db1_id}/backups",
                                   token=user1_token)
        if status in (201, 200):
            t.passed(f"Create backup on pro → {body.get('id', '?')[:8]}")
        else:
            t.failed("Create backup on pro", f"status={status} body={body}")

    # Enable app auth (pro allows more users)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/auth/enable",
                                   token=user1_token)
        if status == 200 and body.get("enabled") is True:
            t.passed("Enable app auth")
        else:
            t.failed("Enable app auth", f"status={status} body={body}")

    # Signup an end-user for the app (public endpoint)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/auth/signup",
                                   body={
                                       "email": f"enduser-{uid}@test.zenith.dev",
                                       "password": f"EndUser-{uid}-Pass!",
                                       "name": "End User",
                                   })
        if status == 200 and "access_token" in body:
            t.passed("App auth signup → end-user registered")
        else:
            t.failed("App auth signup", f"status={status} body={body}")

    # List app auth users
    if app1_id:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/auth/users",
                                   token=user1_token)
        if status == 200 and body.get("total", 0) >= 1:
            t.passed(f"List app auth users → {body['total']} user(s)")
        else:
            t.failed("List app auth users", f"status={status} body={body}")

    # Billing shows pro
    status, body = api_request(base_url, "GET", "/api/v1/billing", token=user1_token)
    if status == 200 and body.get("tier") == "pro":
        t.passed("Billing status → pro tier")
    else:
        t.failed("Billing status after upgrade", f"status={status} tier={body.get('tier')}")

    # ===================================================================
    # G. Cleanup — 7/7
    # ===================================================================
    t.section("7/7", "Cleanup")

    if no_cleanup:
        t.passed("Cleanup skipped (--no-cleanup)")
        return

    cleaned = 0

    # Delete custom domains
    if app1_id:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/domains",
                                   token=user1_token)
        if status == 200 and isinstance(body, list):
            for d in body:
                api_request(base_url, "DELETE",
                            f"/api/v1/apps/{app1_id}/domains/{d['id']}", token=user1_token)
                cleaned += 1

    # Delete storage bucket
    if app1_id and bucket_id:
        api_request(base_url, "DELETE", f"/api/v1/apps/{app1_id}/storage/{bucket_id}",
                    token=user1_token)
        cleaned += 1

    # Delete database
    if app1_id and db1_id:
        api_request(base_url, "DELETE", f"/api/v1/apps/{app1_id}/databases/{db1_id}",
                    token=user1_token)
        cleaned += 1

    # Delete apps
    for aid in [app2_id, app1_id]:
        if aid:
            api_request(base_url, "DELETE", f"/api/v1/apps/{aid}", token=user1_token)
            cleaned += 1

    t.passed(f"Cleaned up {cleaned} resources")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
def main() -> int:
    parser = argparse.ArgumentParser(description="Zenith E2E Smoke Test — Staging")
    parser.add_argument("--verbose", "-v", action="store_true", help="Show failure details")
    parser.add_argument("--no-cleanup", action="store_true", help="Skip resource cleanup")
    parser.add_argument("--base-url", default="", help="API base URL (default: auto port-forward)")
    parser.add_argument("--no-port-forward", action="store_true", help="Skip auto port-forward")
    args = parser.parse_args()

    print()
    print("=" * 50)
    print(f"   {CYAN}Zenith E2E Smoke Test — Staging{NC}")
    print(f"   {time.strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 50)

    base_url = args.base_url or "http://localhost:18080"
    pf_started = False

    # Auto port-forward if no explicit URL and not disabled
    if not args.base_url and not args.no_port_forward:
        print(f"\n{CYAN}Starting port-forward to zenith-api...{NC}")
        if start_port_forward("zenith-staging", "zenith-api", 18080, 8080):
            pf_started = True
            print(f"  Port-forward active → {base_url}")
        else:
            print(f"  {YELLOW}Port-forward failed — assuming API is already reachable at {base_url}{NC}")

    t = TestRunner(verbose=args.verbose)

    try:
        run_tests(base_url, t, args.no_cleanup)
    except KeyboardInterrupt:
        print(f"\n{YELLOW}Interrupted{NC}")
    finally:
        if pf_started:
            stop_port_forward()
            print(f"\n{CYAN}Port-forward stopped{NC}")

    t.summary()
    return 1 if t.fail_count > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
