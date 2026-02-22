## ADDED Requirements

### Requirement: Ansible-Based Platform Deployment
The system SHALL provide Ansible playbooks and roles for deploying the entire Zenith platform (infrastructure + applications) to any k3s server. The same playbooks SHALL work for both staging and production environments via separate inventory files.

#### Scenario: Full platform deployment to staging
- **WHEN** an operator runs `ansible-playbook playbooks/site.yml -i inventory/staging.yml`
- **THEN** the playbook provisions infrastructure (k3s, cert-manager, PostgreSQL, Traefik config), builds and imports Docker images, deploys all Zenith services, configures ingress and TLS

#### Scenario: Full platform deployment to production
- **WHEN** an operator runs `ansible-playbook playbooks/site.yml -i inventory/production.yml`
- **THEN** the same playbook deploys to production using production-specific domains, secrets, and resource limits

#### Scenario: Idempotent re-run
- **WHEN** an operator runs the same playbook twice without code changes
- **THEN** the second run reports no changes (ok/skipped) and does not restart healthy services

### Requirement: Modular Role-Based Deployment
The system SHALL organize deployment into independent Ansible roles that can be executed selectively via tags. Roles include: common, k3s, cert-manager, traefik-config, postgres, keda, monitoring, dns, zenith-build, zenith-import, zenith-namespaces, zenith-api, zenith-landing, zenith-mc, zenith-web, zenith-ingress.

#### Scenario: Deploy only applications
- **WHEN** an operator runs `ansible-playbook playbooks/apps.yml -i inventory/production.yml`
- **THEN** only Zenith application images are built, imported, and deployed — infrastructure components are not touched

#### Scenario: Deploy only infrastructure
- **WHEN** an operator runs `ansible-playbook playbooks/infra.yml -i inventory/staging.yml`
- **THEN** only infrastructure components (k3s, cert-manager, PostgreSQL, KEDA, monitoring) are provisioned or updated

### Requirement: Encrypted Secrets Management
The system SHALL store all deployment secrets (JWT secret, DB passwords, admin credentials, API tokens) in Ansible Vault-encrypted files, one per environment. Secrets are injected into Kubernetes Secrets during deployment.

#### Scenario: Secrets decrypted at deploy time
- **WHEN** an operator runs a playbook with `--ask-vault-pass`
- **THEN** vault-encrypted secrets are decrypted in memory and used to create or update Kubernetes Secrets

#### Scenario: Different secrets per environment
- **WHEN** staging and production vault files contain different JWT secrets and DB passwords
- **THEN** each environment uses its own isolated credentials

### Requirement: Infrastructure Component Installation
The system SHALL install infrastructure components via Helm charts managed by Ansible roles: KEDA (scale-to-zero), cert-manager (TLS), kube-prometheus-stack + Loki (monitoring), and PostgreSQL (StatefulSet). Each component can be enabled/disabled per environment.

#### Scenario: KEDA enabled in production only
- **WHEN** production inventory has `enable_keda: true` and staging has `enable_keda: false`
- **THEN** KEDA is installed only on production

#### Scenario: Monitoring stack installation
- **WHEN** `enable_monitoring: true` is set in the environment
- **THEN** Prometheus, Grafana, Loki, and Promtail are installed via their Helm charts with Zenith-specific values
