<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

# 🧙 CLAUDE.MD - AI AGENT CONFIGURATION

> **⚠️ CRITICAL: This file is identical to `AGENTS.md`.**
> **You MUST read `AGENTS.md` COMPLETELY for all rules and commands.**

---

## 📌 Purpose

This file exists so **Claude CLI** can automatically detect the project context.

**All rules, CLI commands, and architecture guidelines are in `AGENTS.md`.**

---

## 🚨 MANDATORY STEPS

When you open this project:

1. **READ `AGENTS.md` COMPLETELY** - It contains all Lich Framework rules
2. **READ `.lich/rules/ai-behavior.md`** - Lich-first decision logic
3. **READ `agentlog.md`** - Project change history

---

## ⚡ Quick Reference

```bash
# First time setup
lich setup           # Configure AI tools

# Development
lich start           # Start dev environment
lich stop            # Stop everything

# Code Generation (MANDATORY - never write manually)
lich make entity <name>
lich make service <name>
lich make api <name>

# CI (Continuous Integration)
lich ci setup        # Setup act for local CI
lich ci backend      # Backend CI (Docker)
lich ci backend -l   # Backend CI (local, fast)
lich ci web          # Web CI
lich ci admin        # Admin CI
lich ci landing      # Landing CI

# Deploy
lich deploy setup            # Configure SSH, paths
lich deploy stage admin      # Deploy to staging
lich deploy prod backend -v v1.2.3  # Deploy to prod

# Quality
lich test            # Run tests
lich lint --fix      # Fix linting
lich security        # Security scan
lich doctor          # Health check

# Database
lich migration create "msg"
lich migration up
```

---

## 📚 Files to Read

| Priority | File | Purpose |
|----------|------|---------|
| 🔴 HIGH | `AGENTS.md` | **Master AI prompt - READ FIRST** |
| 🔴 HIGH | `agentlog.md` | Change history - ALWAYS UPDATE |
| 🔴 HIGH | `.lich/rules/ai-behavior.md` | Lich-first decision logic |
| 🟡 MED | `.lich/rules/backend.md` | Backend architecture |
| 🟡 MED | `.lich/rules/frontend.md` | Frontend architecture |
| 🟡 MED | `.lich/rules/ui-ux.md` | UI/UX design rules |
| 🟢 LOW | `.lich/workflows/` | Step-by-step guides |

---

## 🔗 Reminder

**AGENTS.md = CLAUDE.md**

Both files enforce the same rules. If you're reading this, go read `AGENTS.md`.

**🧙 Meta Architect Activated.**
