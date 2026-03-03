#!/usr/bin/env python3
"""
Zenith Platform E2E Smoke Test — Staging
=========================================
Exercises the full user lifecycle against the staging API:
  health → auth → free tier → limits → IDOR → pro upgrade →
  admin → sessions/api-keys → compliance → cleanup

Usage:
    python3 infra/scripts/e2e-smoke-staging.py [--verbose] [--no-cleanup] [--base-url URL] [--no-port-forward]

Prerequisites (if no --base-url given):
    kubectl port-forward svc/zenith-api 18080:8080 -n zenith-staging

Returns exit 0 if all tests pass, exit 1 if any fail.
"""

import argparse
import os
import sys
import time

from e2e_lib import (
    CYAN,
    NC,
    YELLOW,
    TestRunner,
    api_request,
    print_banner,
    rand_id,
    start_port_forward,
    stop_port_forward,
)

# Admin credentials (from environment or defaults for staging)
ADMIN_EMAIL = os.environ.get("ZENITH_ADMIN_EMAIL", "admin@freezenith.com")
ADMIN_PASSWORD = os.environ.get("ZENITH_ADMIN_PASSWORD", "8i3wIotgaZEgxVnXMEpA")


def run_tests(base_url: str, t: TestRunner, no_cleanup: bool) -> None:
    uid = rand_id()

    # Stash IDs for cleanup
    user1_token = ""
    user2_token = ""
    admin_token = ""
    app1_id = ""
    app2_id = ""
    db1_id = ""
    bucket1_id = ""
    bucket2_id = ""
    api_key_id = ""

    # ==================================================================
    # 1/10 — Health & Version
    # ==================================================================
    t.section("1/10", "Health & Version")

    status, body = api_request(base_url, "GET", "/health")
    if status == 200:
        t.passed("Health check -> 200")
    else:
        t.failed("Health check -> 200", f"got {status}")
        print(f"\n  {YELLOW}API unreachable at {base_url}/health — aborting.{NC}\n")
        return

    status, body = api_request(base_url, "GET", "/api/v1/version")
    if status == 200 and isinstance(body, dict):
        ver = body.get("version", "?")
        t.passed(f"Version endpoint -> {ver}")
    else:
        t.failed("Version endpoint", f"status={status}")

    # ==================================================================
    # 2/10 — Auth
    # ==================================================================
    t.section("2/10", "Auth — Register, Login, Refresh, JWT Checks")

    email1 = f"smoke-{uid}-1@test.zenith.dev"
    email2 = f"smoke-{uid}-2@test.zenith.dev"
    password = f"Sm0ke-{uid}-Pass!"

    # Register user 1
    status, body = api_request(base_url, "POST", "/api/v1/auth/register", body={
        "email": email1, "password": password, "name": f"Smoke {uid}",
    })
    if status in (200, 201) and "access_token" in body:
        user1_token = body["access_token"]
        t.passed(f"Register user1 ({email1})")
    else:
        t.failed("Register user1", f"status={status} body={body}")
        return  # can't continue without auth

    # Login user 1
    status, body = api_request(base_url, "POST", "/api/v1/auth/login", body={
        "email": email1, "password": password,
    })
    if status == 200 and "access_token" in body:
        user1_token = body["access_token"]
        refresh_token = body.get("refresh_token", "")
        t.passed("Login user1")
    else:
        t.failed("Login user1", f"status={status}")

    # Refresh token
    if refresh_token:
        status, body = api_request(base_url, "POST", "/api/v1/auth/refresh", body={
            "refresh_token": refresh_token,
        })
        if status == 200 and "access_token" in body:
            user1_token = body["access_token"]
            t.passed("Refresh token")
        else:
            t.failed("Refresh token", f"status={status}")
    else:
        t.skipped("Refresh token", "no refresh_token in login response")

    # Invalid JWT -> 401
    status, _ = api_request(base_url, "GET", "/api/v1/apps", token="invalid.jwt.token")
    if status == 401:
        t.passed("Invalid JWT -> 401")
    else:
        t.failed("Invalid JWT should be 401", f"got {status}")

    # No JWT -> 401
    status, _ = api_request(base_url, "GET", "/api/v1/apps")
    if status == 401:
        t.passed("No JWT -> 401")
    else:
        t.failed("No JWT should be 401", f"got {status}")

    # Verify free plan
    status, body = api_request(base_url, "GET", "/api/v1/plan", token=user1_token)
    if status == 200 and body.get("tier") == "free":
        limits = body.get("limits", {})
        t.passed(f"Free plan assigned (max_apps={limits.get('max_apps')})")
    else:
        t.failed("Free plan check", f"status={status} tier={body.get('tier') if isinstance(body, dict) else body}")

    # Billing status
    status, body = api_request(base_url, "GET", "/api/v1/billing", token=user1_token)
    if status == 200 and body.get("tier") == "free":
        t.passed("Billing status -> free tier")
    else:
        t.failed("Billing status", f"status={status}")

    # ==================================================================
    # 3/10 — Free Tier: Create Resources
    # ==================================================================
    t.section("3/10", "Free Tier — Create Resources")

    # Create app 1
    status, body = api_request(base_url, "POST", "/api/v1/apps", token=user1_token, body={
        "name": f"smoke-app-{uid}", "repo_url": "https://github.com/test/repo",
    })
    if status == 201 and body.get("id"):
        app1_id = body["id"]
        t.passed(f"Create app1 -> {body.get('name')} (id={app1_id[:8]})")
    else:
        t.failed("Create app1", f"status={status} body={body}")

    # Create database 1
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/databases",
                                   token=user1_token, body={"name": f"smoke-db-{uid}", "engine": "postgresql"})
        if status == 201 and body.get("id"):
            db1_id = body["id"]
            t.passed(f"Create db1 -> {body.get('name')} (engine={body.get('engine')})")
        else:
            t.failed("Create db1", f"status={status} body={body}")

    # Create storage bucket 1
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/storage",
                                   token=user1_token, body={"name": f"smoke-bucket1-{uid}"})
        if status == 201 and body.get("id"):
            bucket1_id = body["id"]
            t.passed(f"Create bucket1 -> {body.get('name')}")
        else:
            t.failed("Create bucket1", f"status={status} body={body}")

    # Register a release
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
            t.passed(f"Create release -> {release_id[:8]}")
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
            t.passed(f"Deploy release -> deployment={deployment_id[:8]}")
        else:
            t.failed("Deploy release", f"status={status} body={body}")

    # List deployments
    if app1_id:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/deployments",
                                   token=user1_token)
        if status == 200 and body.get("total", 0) >= 1:
            t.passed(f"List deployments -> {body['total']} found")
        else:
            t.failed("List deployments", f"status={status} total={body.get('total') if isinstance(body, dict) else body}")

    # ==================================================================
    # 4/10 — Free Tier Limits
    # ==================================================================
    t.section("4/10", "Free Tier — Enforce Limits")

    # 2nd app -> 403
    if app1_id:
        status, _ = api_request(base_url, "POST", "/api/v1/apps", token=user1_token, body={
            "name": f"smoke-app2-{uid}", "repo_url": "https://github.com/test/repo2",
        })
        if status == 403:
            t.passed("2nd app -> 403 (plan limit enforced)")
        else:
            t.failed("2nd app should be 403", f"got {status}")

    # Custom domain -> 403
    if app1_id:
        status, _ = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/domains",
                                token=user1_token, body={"domain": "custom.example.com"})
        if status == 403:
            t.passed("Custom domain -> 403 (free tier)")
        else:
            t.failed("Custom domain should be 403", f"got {status}")

    # 2nd database -> 403
    if app1_id:
        status, _ = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/databases",
                                token=user1_token, body={"name": f"smoke-db2-{uid}", "engine": "postgresql"})
        if status == 403:
            t.passed("2nd database -> 403 (plan limit enforced)")
        else:
            t.failed("2nd database should be 403", f"got {status}")

    # 2nd storage bucket -> 403
    if app1_id:
        status, _ = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/storage",
                                token=user1_token, body={"name": f"smoke-bucket2-{uid}"})
        if status == 403:
            t.passed("2nd bucket -> 403 (plan limit enforced)")
        else:
            t.failed("2nd bucket should be 403", f"got {status}")

    # Backup -> 403
    if app1_id and db1_id:
        status, _ = api_request(base_url, "POST",
                                f"/api/v1/apps/{app1_id}/databases/{db1_id}/backups",
                                token=user1_token)
        if status == 403:
            t.passed("Backup -> 403 (free tier, Pro required)")
        else:
            t.failed("Backup should be 403 on free tier", f"got {status}")

    # ==================================================================
    # 5/10 — Security / IDOR
    # ==================================================================
    t.section("5/10", "Security — IDOR & Auth")

    # Register user 2
    status, body = api_request(base_url, "POST", "/api/v1/auth/register", body={
        "email": email2, "password": password, "name": f"Smoke2 {uid}",
    })
    if status in (200, 201) and "access_token" in body:
        user2_token = body["access_token"]
        t.passed(f"Register user2 ({email2})")
    else:
        t.failed("Register user2", f"status={status}")

    # User2 -> user1's app (GET)
    if app1_id and user2_token:
        status, _ = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}", token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 GET app1 = {status}")
        else:
            t.failed("IDOR: user2 should not access app1", f"got {status}")

    # User2 -> delete user1's app
    if app1_id and user2_token:
        status, _ = api_request(base_url, "DELETE", f"/api/v1/apps/{app1_id}", token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 DELETE app1 = {status}")
        else:
            t.failed("IDOR: user2 should not delete app1", f"got {status}")

    # User2 -> user1's databases
    if app1_id and user2_token:
        status, _ = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/databases", token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 GET app1 databases = {status}")
        else:
            t.failed("IDOR: user2 should not list app1 databases", f"got {status}")

    # User2 -> user1's storage
    if app1_id and user2_token:
        status, _ = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/storage", token=user2_token)
        if status in (403, 404):
            t.passed(f"IDOR: user2 GET app1 storage = {status}")
        else:
            t.failed("IDOR: user2 should not list app1 storage", f"got {status}")

    # ==================================================================
    # 6/10 — Pro Upgrade & Pro Features
    # ==================================================================
    t.section("6/10", "Pro Upgrade & Pro Features")

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
            t.passed("Pro limits correct (max_apps=5, custom_domain=true)")
        else:
            t.failed("Pro limits", f"limits={limits}")
    else:
        t.failed("Pro plan check", f"status={status}")

    # 2nd app (should succeed on pro)
    status, body = api_request(base_url, "POST", "/api/v1/apps", token=user1_token, body={
        "name": f"smoke-app2-{uid}", "repo_url": "https://github.com/test/repo2",
    })
    if status == 201 and body.get("id"):
        app2_id = body["id"]
        t.passed(f"Create app2 on pro -> {body.get('name')}")
    else:
        t.failed("Create app2 on pro", f"status={status} body={body}")

    # Custom domain (pro allows)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/domains",
                                   token=user1_token, body={"domain": f"smoke-{uid}.example.com"})
        if status == 201:
            t.passed(f"Add custom domain -> {body.get('domain') if isinstance(body, dict) else '?'}")
        else:
            t.failed("Add custom domain on pro", f"status={status} body={body}")

    # Backup (pro allows)
    if app1_id and db1_id:
        status, body = api_request(base_url, "POST",
                                   f"/api/v1/apps/{app1_id}/databases/{db1_id}/backups",
                                   token=user1_token)
        if status in (200, 201):
            t.passed(f"Create backup on pro -> {body.get('id', '?')[:8] if isinstance(body, dict) else '?'}")
        else:
            t.failed("Create backup on pro", f"status={status} body={body}")

    # Storage bucket 2 (pro allows more)
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/storage",
                                   token=user1_token, body={"name": f"smoke-bucket2-{uid}"})
        if status == 201 and body.get("id"):
            bucket2_id = body["id"]
            t.passed(f"Create bucket2 on pro -> {body.get('name')}")
        else:
            t.failed("Create bucket2 on pro", f"status={status} body={body}")

    # Enable app auth
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/auth/enable",
                                   token=user1_token)
        if status == 200 and isinstance(body, dict) and body.get("enabled") is True:
            t.passed("Enable app auth")
        else:
            t.failed("Enable app auth", f"status={status} body={body}")

    # Signup end-user for the app
    if app1_id:
        status, body = api_request(base_url, "POST", f"/api/v1/apps/{app1_id}/auth/signup",
                                   body={
                                       "email": f"enduser-{uid}@test.zenith.dev",
                                       "password": f"EndUser-{uid}-Pass!",
                                       "name": "End User",
                                   })
        if status == 200 and "access_token" in body:
            t.passed("App auth signup -> end-user registered")
        else:
            t.failed("App auth signup", f"status={status} body={body}")

    # List app auth users
    if app1_id:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/auth/users",
                                   token=user1_token)
        if status == 200 and body.get("total", 0) >= 1:
            t.passed(f"List app auth users -> {body['total']} user(s)")
        else:
            t.failed("List app auth users", f"status={status} body={body}")

    # Billing shows pro
    status, body = api_request(base_url, "GET", "/api/v1/billing", token=user1_token)
    if status == 200 and body.get("tier") == "pro":
        t.passed("Billing status -> pro tier")
    else:
        t.failed("Billing status after upgrade", f"status={status} tier={body.get('tier') if isinstance(body, dict) else body}")

    # ==================================================================
    # 7/10 — Admin Endpoints
    # ==================================================================
    t.section("7/10", "Admin Endpoints")

    # Login as admin
    status, body = api_request(base_url, "POST", "/api/v1/auth/login", body={
        "email": ADMIN_EMAIL, "password": ADMIN_PASSWORD,
    })
    if status == 200 and "access_token" in body:
        admin_token = body["access_token"]
        t.passed(f"Admin login ({ADMIN_EMAIL})")
    else:
        t.failed("Admin login", f"status={status} body={body}")

    if admin_token:
        # Dashboard stats
        status, body = api_request(base_url, "GET", "/api/v1/admin/dashboard/stats", token=admin_token)
        if status == 200 and isinstance(body, dict):
            t.passed(f"Dashboard stats -> {body.get('total_users', '?')} users, {body.get('total_apps', '?')} apps")
        else:
            t.failed("Dashboard stats", f"status={status}")

        # Platform usage
        status, body = api_request(base_url, "GET", "/api/v1/admin/dashboard/usage", token=admin_token)
        if status == 200:
            t.passed("Dashboard usage")
        else:
            t.failed("Dashboard usage", f"status={status}")

        # List customers
        status, body = api_request(base_url, "GET", "/api/v1/admin/customers", token=admin_token)
        if status == 200:
            count = body.get("total", len(body)) if isinstance(body, dict) else len(body) if isinstance(body, list) else "?"
            t.passed(f"List customers -> {count}")
        else:
            t.failed("List customers", f"status={status}")

        # Customer stats
        status, body = api_request(base_url, "GET", "/api/v1/admin/customers/stats", token=admin_token)
        if status == 200:
            t.passed("Customer stats")
        else:
            t.failed("Customer stats", f"status={status}")

        # Audit log
        status, body = api_request(base_url, "GET", "/api/v1/admin/audit", token=admin_token)
        if status == 200:
            count = body.get("total", "?") if isinstance(body, dict) else "?"
            t.passed(f"Audit log -> {count} entries")
        else:
            t.failed("Audit log", f"status={status}")

        # Clusters
        status, body = api_request(base_url, "GET", "/api/v1/admin/clusters", token=admin_token)
        if status == 200:
            t.passed("List clusters")
        else:
            t.failed("List clusters", f"status={status}")

        # Modules
        status, body = api_request(base_url, "GET", "/api/v1/admin/modules", token=admin_token)
        if status == 200:
            count = len(body) if isinstance(body, list) else body.get("total", "?") if isinstance(body, dict) else "?"
            t.passed(f"List modules -> {count}")
        else:
            t.failed("List modules", f"status={status}")

        # Settings
        status, body = api_request(base_url, "GET", "/api/v1/admin/settings", token=admin_token)
        if status == 200:
            t.passed("Get settings")
        else:
            t.failed("Get settings", f"status={status}")

        # Infrastructure
        status, body = api_request(base_url, "GET", "/api/v1/admin/infrastructure", token=admin_token)
        if status == 200:
            t.passed("Infrastructure overview")
        else:
            t.failed("Infrastructure overview", f"status={status}")

        # Platform state
        status, body = api_request(base_url, "GET", "/api/v1/admin/state", token=admin_token)
        if status == 200:
            t.passed("Platform state")
        else:
            t.failed("Platform state", f"status={status}")

        # Plans
        status, body = api_request(base_url, "GET", "/api/v1/admin/plans", token=admin_token)
        if status == 200:
            t.passed("List plans")
        else:
            t.failed("List plans", f"status={status}")

        # Updates check
        status, body = api_request(base_url, "GET", "/api/v1/admin/updates/check", token=admin_token)
        if status == 200:
            t.passed("Check updates")
        else:
            t.failed("Check updates", f"status={status}")

        # Admin billing overview
        status, body = api_request(base_url, "GET", "/api/v1/admin/billing/overview", token=admin_token)
        if status == 200:
            t.passed("Admin billing overview")
        else:
            t.failed("Admin billing overview", f"status={status}")

        # Non-admin should not access admin endpoints
        status, _ = api_request(base_url, "GET", "/api/v1/admin/dashboard/stats", token=user1_token)
        if status == 403:
            t.passed("Non-admin -> admin endpoint = 403")
        else:
            t.failed("Non-admin should get 403 on admin endpoint", f"got {status}")

    else:
        t.skipped("Admin endpoints", "admin login failed")

    # ==================================================================
    # 8/10 — Sessions & API Keys
    # ==================================================================
    t.section("8/10", "Sessions & API Keys")

    # List sessions
    status, body = api_request(base_url, "GET", "/api/v1/auth/sessions", token=user1_token)
    if status == 200:
        count = body.get("total", len(body)) if isinstance(body, dict) else len(body) if isinstance(body, list) else 0
        t.passed(f"List sessions -> {count}")
    else:
        t.failed("List sessions", f"status={status}")

    # Create API key
    status, body = api_request(base_url, "POST", "/api/v1/api-keys", token=user1_token, body={
        "name": f"smoke-key-{uid}",
    })
    if status in (200, 201) and isinstance(body, dict):
        api_key_id = body.get("id", "")
        raw_key = body.get("key", body.get("api_key", ""))
        t.passed(f"Create API key -> {api_key_id[:8] if api_key_id else '?'}")
    else:
        t.failed("Create API key", f"status={status} body={body}")

    # List API keys
    status, body = api_request(base_url, "GET", "/api/v1/api-keys", token=user1_token)
    if status == 200:
        count = body.get("total", len(body)) if isinstance(body, dict) else len(body) if isinstance(body, list) else 0
        t.passed(f"List API keys -> {count}")
    else:
        t.failed("List API keys", f"status={status}")

    # Delete API key
    if api_key_id:
        status, _ = api_request(base_url, "DELETE", f"/api/v1/api-keys/{api_key_id}", token=user1_token)
        if status in (200, 204):
            t.passed("Delete API key")
        else:
            t.failed("Delete API key", f"status={status}")

    # ==================================================================
    # 9/10 — Compliance & Settings
    # ==================================================================
    t.section("9/10", "Compliance & Settings")

    # MFA status
    status, body = api_request(base_url, "GET", "/api/v1/auth/mfa", token=user1_token)
    if status == 200:
        enabled = body.get("enabled", False) if isinstance(body, dict) else False
        t.passed(f"MFA status -> enabled={enabled}")
    else:
        t.failed("MFA status", f"status={status}")

    # Compliance status
    status, body = api_request(base_url, "GET", "/api/v1/compliance", token=user1_token)
    if status == 200:
        t.passed("Compliance status")
    else:
        t.failed("Compliance status", f"status={status}")

    # Branding (GET — may be 403 for non-enterprise, that's fine)
    status, body = api_request(base_url, "GET", "/api/v1/settings/branding", token=user1_token)
    if status == 200:
        t.passed("Get branding config")
    elif status == 403:
        t.passed("Branding -> 403 (enterprise only, expected)")
    else:
        t.failed("Branding", f"status={status}")

    # DPA status (GET — may be 403 for non-team, that's fine)
    status, body = api_request(base_url, "GET", "/api/v1/settings/dpa", token=user1_token)
    if status == 200:
        t.passed("DPA status")
    elif status == 403:
        t.passed("DPA -> 403 (team+ only, expected)")
    else:
        t.failed("DPA status", f"status={status}")

    # SSO list (GET — may be 403 for non-team)
    status, body = api_request(base_url, "GET", "/api/v1/settings/sso", token=user1_token)
    if status == 200:
        t.passed("List SSO configs")
    elif status == 403:
        t.passed("SSO -> 403 (team+ only, expected)")
    else:
        t.failed("SSO list", f"status={status}")

    # IP whitelist (GET — may be 403 for non-enterprise)
    status, body = api_request(base_url, "GET", "/api/v1/settings/ip-whitelist", token=user1_token)
    if status == 200:
        t.passed("IP whitelist")
    elif status == 403:
        t.passed("IP whitelist -> 403 (enterprise only, expected)")
    else:
        t.failed("IP whitelist", f"status={status}")

    # ==================================================================
    # 10/10 — Cleanup
    # ==================================================================
    t.section("10/10", "Cleanup")

    if no_cleanup:
        t.passed("Cleanup skipped (--no-cleanup)")
        return

    cleaned = 0

    # Delete custom domains
    if app1_id:
        status, body = api_request(base_url, "GET", f"/api/v1/apps/{app1_id}/domains", token=user1_token)
        if status == 200 and isinstance(body, list):
            for d in body:
                api_request(base_url, "DELETE",
                            f"/api/v1/apps/{app1_id}/domains/{d['id']}", token=user1_token)
                cleaned += 1

    # Delete storage buckets
    for bid in [bucket2_id, bucket1_id]:
        if app1_id and bid:
            api_request(base_url, "DELETE", f"/api/v1/apps/{app1_id}/storage/{bid}", token=user1_token)
            cleaned += 1

    # Delete database
    if app1_id and db1_id:
        api_request(base_url, "DELETE", f"/api/v1/apps/{app1_id}/databases/{db1_id}", token=user1_token)
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

    print_banner("Zenith E2E Smoke Test — Staging")

    base_url = args.base_url or "http://localhost:18080"
    pf_started = False

    # Auto port-forward if no explicit URL and not disabled
    if not args.base_url and not args.no_port_forward:
        print(f"\n{CYAN}Starting port-forward to zenith-api...{NC}")
        if start_port_forward("zenith-staging", "zenith-api", 18080, 8080):
            pf_started = True
            print(f"  Port-forward active -> {base_url}")
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
