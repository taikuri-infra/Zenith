"""
Zenith E2E Test Library — Shared Helpers
=========================================
Common utilities used by both e2e-infra-staging.py and e2e-smoke-staging.py.

Provides:
  - ANSI color constants
  - TestRunner (pass/fail/section/summary)
  - api_request() HTTP helper
  - rand_id() random identifier
  - Port-forward start/stop
  - DNS, SSL, HTTPS check helpers (for infra tests)
"""

import json
import random
import socket
import ssl
import string
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Optional

# ---------------------------------------------------------------------------
# ANSI colours
# ---------------------------------------------------------------------------
RED = "\033[0;31m"
GREEN = "\033[0;32m"
YELLOW = "\033[1;33m"
BLUE = "\033[0;34m"
CYAN = "\033[0;36m"
BOLD = "\033[1m"
NC = "\033[0m"


# ---------------------------------------------------------------------------
# Test runner
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

    def skipped(self, label: str, reason: str = "") -> None:
        self.total += 1
        msg = f"  {YELLOW}SKIP{NC} {label}"
        if reason:
            msg += f" ({reason})"
        print(msg)

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
                print(f"    {RED}x{NC} {f}")
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
    timeout: int = 15,
) -> tuple:
    """Make an HTTP request and return (status_code, parsed_json_or_text)."""
    url = f"{base}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", f"Bearer {token}")

    try:
        resp = urllib.request.urlopen(req, timeout=timeout)
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
# Port-forward helpers
# ---------------------------------------------------------------------------
_PF_PROC: Optional[subprocess.Popen] = None


def start_port_forward(namespace: str, svc: str, local_port: int, remote_port: int) -> bool:
    global _PF_PROC
    try:
        _PF_PROC = subprocess.Popen(
            ["kubectl", "port-forward", f"svc/{svc}", f"{local_port}:{remote_port}", "-n", namespace],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        time.sleep(2)
        return _PF_PROC.poll() is None
    except FileNotFoundError:
        return False


def stop_port_forward() -> None:
    global _PF_PROC
    if _PF_PROC and _PF_PROC.poll() is None:
        _PF_PROC.terminate()
        _PF_PROC.wait(timeout=5)
        _PF_PROC = None


# ---------------------------------------------------------------------------
# DNS helper
# ---------------------------------------------------------------------------
def check_dns(domain: str, expected_ip: Optional[str] = None) -> tuple:
    """Resolve domain and return (resolved_ip, ok).

    If expected_ip is given, ok means it matched.
    If expected_ip is None, ok means *any* resolution succeeded.
    """
    try:
        resolved = socket.gethostbyname(domain)
    except socket.gaierror:
        return ("", False)
    if expected_ip is None:
        return (resolved, True)
    return (resolved, resolved == expected_ip)


# ---------------------------------------------------------------------------
# SSL helper
# ---------------------------------------------------------------------------
def check_ssl_cert(domain: str, port: int = 443) -> dict:
    """Connect via TLS and return cert info dict.

    Returns:
        {
            "ok": bool,
            "subject": str,
            "issuer": str,
            "not_after": str,          # ISO date
            "days_remaining": int,
            "error": str,
        }
    """
    result = {"ok": False, "self_signed": False, "subject": "", "issuer": "", "not_after": "", "days_remaining": -1, "error": ""}

    def _read_cert(ctx):
        with socket.create_connection((domain, port), timeout=10) as sock:
            with ctx.wrap_socket(sock, server_hostname=domain) as ssock:
                return ssock.getpeercert()

    def _parse_cert(cert):
        for rdn in cert.get("subject", ()):
            for attr, val in rdn:
                if attr == "commonName":
                    result["subject"] = val
        for rdn in cert.get("issuer", ()):
            for attr, val in rdn:
                if attr == "organizationName":
                    result["issuer"] = val
        not_after_str = cert.get("notAfter", "")
        if not_after_str:
            expiry = datetime.strptime(not_after_str, "%b %d %H:%M:%S %Y %Z").replace(tzinfo=timezone.utc)
            result["not_after"] = expiry.strftime("%Y-%m-%d")
            result["days_remaining"] = (expiry - datetime.now(timezone.utc)).days

    # Try with full verification first
    try:
        cert = _read_cert(ssl.create_default_context())
        if cert:
            _parse_cert(cert)
            result["ok"] = True
            return result
        result["error"] = "no certificate returned"
        return result
    except ssl.SSLCertVerificationError:
        pass  # fall through to unverified attempt
    except Exception as e:
        result["error"] = str(e)
        return result

    # Fallback: self-signed / untrusted — still read the cert
    try:
        ctx = ssl.create_default_context()
        ctx.check_hostname = False
        ctx.verify_mode = ssl.CERT_NONE
        with socket.create_connection((domain, port), timeout=10) as sock:
            with ctx.wrap_socket(sock, server_hostname=domain) as ssock:
                der = ssock.getpeercert(binary_form=True)
                if der:
                    # Decode DER to get expiry via openssl-style parsing
                    cert = ssl.DER_cert_to_PEM_cert(der)
                    result["ok"] = True
                    result["self_signed"] = True
                    result["error"] = "self-signed or untrusted CA"
                else:
                    result["error"] = "no certificate returned (unverified)"
    except Exception as e:
        result["error"] = f"unverified attempt failed: {e}"

    return result


# ---------------------------------------------------------------------------
# HTTPS helper
# ---------------------------------------------------------------------------
def check_https_status(url: str, follow_redirects: bool = True, timeout: int = 10) -> tuple:
    """Fetch URL and return (status_code, body_snippet).

    With follow_redirects=False, returns the redirect status (301/308) instead of following.
    """
    req = urllib.request.Request(url, method="GET")
    req.add_header("User-Agent", "ZenithE2E/1.0")

    if not follow_redirects:
        # Use a custom opener that doesn't follow redirects
        class NoRedirectHandler(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, headers, newurl):
                raise urllib.error.HTTPError(newurl, code, msg, headers, fp)

        opener = urllib.request.build_opener(NoRedirectHandler)
        try:
            resp = opener.open(req, timeout=timeout)
            return resp.status, resp.read().decode()[:4000]
        except urllib.error.HTTPError as e:
            return e.code, ""
        except Exception as e:
            return 0, str(e)

    try:
        resp = urllib.request.urlopen(req, timeout=timeout)
        return resp.status, resp.read().decode()[:4000]
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode()[:4000] if e.fp else ""
    except Exception as e:
        return 0, str(e)


# ---------------------------------------------------------------------------
# SSH helper (for K8s checks)
# ---------------------------------------------------------------------------
def ssh_command(host: str, cmd: str, timeout: int = 15) -> tuple:
    """Run a command on remote host via SSH. Returns (ok, stdout)."""
    try:
        result = subprocess.run(
            ["ssh", "-o", "ConnectTimeout=5", "-o", "StrictHostKeyChecking=no", host, cmd],
            capture_output=True, text=True, timeout=timeout,
        )
        return result.returncode == 0, result.stdout.strip()
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False, ""


# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
def print_banner(title: str) -> None:
    print()
    print("=" * 50)
    print(f"   {CYAN}{title}{NC}")
    print(f"   {time.strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 50)
