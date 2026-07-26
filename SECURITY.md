# Security Policy

FreeZenith is a self-hostable platform that runs other people's applications, so
we take security seriously and appreciate responsible disclosure.

## Reporting a vulnerability

**Please do not open a public issue for security problems.**

Report privately through GitHub's built-in advisory flow:

1. Go to the repository's **Security** tab → **Report a vulnerability**
   (**[Advisories › Report a vulnerability](https://github.com/taikuri-infra/Zenith/security/advisories/new)**).
2. Describe the issue, affected version/commit, and steps to reproduce.

If you cannot use GitHub advisories, email **security@freezenith.com** with the
same details.

## What to expect

- **Acknowledgement:** within 3 business days.
- **Assessment & triage:** we will confirm the issue and give you an estimated
  timeline for a fix.
- **Fix & disclosure:** once a fix is available we will coordinate a disclosure
  date with you and credit you (if you wish) in the release notes.

## Scope

- The FreeZenith platform code in this repository (API, web, CLI, installer,
  Helm charts, and the self-host Docker Compose stack).
- The published install script and container images under
  `ghcr.io/taikuri-infra/`.

Out of scope: vulnerabilities in third-party dependencies (report those
upstream), and issues that require a already-compromised host or physical
access.

## Supported versions

FreeZenith is pre-1.0 and ships from `main`. Security fixes land on `main` and
the latest published images; please run the current release before reporting.
