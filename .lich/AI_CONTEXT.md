# AI Context - moneyFactory

> This file contains all context an AI needs to work with this project.

## Project Identity

```yaml
name: moneyFactory
slug: moneyfactory
type: saas_platform
description: A brief description of the project
author: Your Name
email: your@email.com
```

## Configuration

```yaml
authentication: jwt_builtin
database: postgresql
cache: yes
i18n: yes
default_language: en
tls: yes
domain: localhost
structured_logging: yes
landing_backend: wordpress_api
temporal: yes
```

## Architecture Type

This is a **saas_platform** project with:
- Multi-tenant architecture
- Subscription management
- User dashboards
- Admin panel

## What I Need to Know

### Backend Rules
- Follow Lich Architecture (`.lich/rules/backend.md`)
- Entities are pure domain models
- Services contain business logic
- Ports are interfaces, Adapters are implementations

### Frontend Rules
- Use CSS Modules (no Tailwind)
- TypeScript strict mode
- Component-per-file
- RTL support for en

### Security Rules
- See `.lich/rules/security.md`
- No localStorage for tokens
- Validate ALL inputs
- Non-root Docker containers

## When Making Changes

1. Follow the appropriate rule file in `.lich/rules/`
2. Write tests for new code
3. Update documentation
4. Update `agentlog.md` with WHAT, WHY, WHEN

## Available Workflows

See `.lich/workflows/` for:
- add-feature.md
- add-entity.md
- add-api.md
- create-landing.md
