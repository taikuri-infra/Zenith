# 18 — Kyverno Policies & Falco Runtime Security

> **Purpose:** Understand how admission policies prevent dangerous workloads from running and how runtime security detects anomalous behavior.
> **Audience:** Any developer who needs to understand why a deployment was rejected, add new policies, or investigate security alerts.
> **Last Updated:** 2026-03-03
> **Related:** [03-phase3-cluster-bootstrap.md](./03-phase3-cluster-bootstrap.md) (installation steps), [06-security-model.md](./06-security-model.md) (full security threat model), [08-observability.md](./08-observability.md) (Falco alerts in Grafana)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose Them](#2-why-we-chose-them)
3. [Architecture Diagram](#3-architecture-diagram)
4. [Kyverno — Admission Policies](#4-kyverno--admission-policies)
5. [How Kyverno Blocks a Bad Deployment](#5-how-kyverno-blocks-a-bad-deployment)
6. [Falco — Runtime Security](#6-falco--runtime-security)
7. [How Falco Detects an Attack](#7-how-falco-detects-an-attack)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

Zenith uses **two security layers** that work together:

- **Kyverno** = **Prevention** — Blocks bad resources BEFORE they're created (admission webhook)
- **Falco** = **Detection** — Detects suspicious activity AFTER containers are running (eBPF monitoring)

```
Security timeline:

  kubectl apply (deploy)          Container running               Attack detected
        │                              │                              │
        ▼                              ▼                              ▼
   ┌──────────┐                  ┌──────────┐                   ┌──────────┐
   │ KYVERNO  │                  │ Falco    │                   │ Falco    │
   │ Blocks   │                  │ Monitors │                   │ Alerts   │
   │ unsigned │                  │ syscalls │                   │ via      │
   │ images,  │                  │ via eBPF │                   │ Sidekick │
   │ missing  │                  │          │                   │          │
   │ labels   │                  │          │                   │          │
   └──────────┘                  └──────────┘                   └──────────┘
   ADMISSION TIME                RUNTIME                         ALERT
```

---

## 2. Why We Chose Them

| Feature | Kyverno | OPA/Gatekeeper | Pod Security Admission |
|---------|---------|---------------|----------------------|
| Policy language | YAML (easy!) | Rego (hard to learn) | Labels only |
| Mutating policies | Yes | Yes | No |
| Generate resources | Yes (auto-create) | No | No |
| Image verification | Built-in (cosign) | External | No |
| Learning curve | Low (K8s-native YAML) | High (Rego language) | Very low |

| Feature | Falco | Tetragon | Sysdig |
|---------|-------|----------|--------|
| eBPF-based | Yes | Yes | Yes (commercial) |
| Pre-built rules | 100+ | Fewer | Many |
| Alert forwarding | Falcosidekick (30+ outputs) | Limited | Built-in |
| Community | Large (CNCF) | Growing | Commercial |
| Cost | Free | Free | $$$ |

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SECURITY ENFORCEMENT IN ZENITH                           │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    KYVERNO (kyverno namespace)                         │  │
│  │                    Admission Webhook Controller                        │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ How it works:                                                    │ │  │
│  │  │                                                                  │ │  │
│  │  │  kubectl apply ──▶ K8s API Server ──▶ Kyverno Webhook ──▶ Allow │ │  │
│  │  │  (create/update)   (admission hook)   (check policies)    or    │ │  │
│  │  │                                                           Deny  │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                       │  │
│  │  Policy Types:                                                        │  │
│  │  ┌───────────────┐  ┌───────────────┐  ┌─────────────────────────┐   │  │
│  │  │ VALIDATE      │  │ MUTATE        │  │ GENERATE                │   │  │
│  │  │ Block bad     │  │ Fix resources │  │ Auto-create resources   │   │  │
│  │  │ resources     │  │ before saving │  │ when trigger fires      │   │  │
│  │  │               │  │               │  │                         │   │  │
│  │  │ e.g. Block    │  │ e.g. Add      │  │ e.g. Auto-create       │   │  │
│  │  │ unsigned      │  │ default       │  │ NetworkPolicy when     │   │  │
│  │  │ images        │  │ resource      │  │ new namespace created  │   │  │
│  │  │               │  │ limits        │  │                         │   │  │
│  │  └───────────────┘  └───────────────┘  └─────────────────────────┘   │  │
│  │                                                                       │  │
│  │  Resources: 100m-500m CPU, 256Mi-512Mi RAM                           │  │
│  │  Replicas: 1                                                          │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    FALCO (falco namespace)                             │  │
│  │                    Runtime Security DaemonSet                          │  │
│  │                                                                       │  │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │  │
│  │  │ How it works:                                                    │ │  │
│  │  │                                                                  │ │  │
│  │  │  Container  ──▶  Linux Kernel  ──▶  Falco eBPF  ──▶  Match     │ │  │
│  │  │  makes a         syscall            program          against    │ │  │
│  │  │  syscall         (open, exec,       hooks into       rules     │ │  │
│  │  │                   connect...)       kernel                     │ │  │
│  │  │                                                      │          │ │  │
│  │  │                                                      ▼          │ │  │
│  │  │                                                ┌──────────┐    │ │  │
│  │  │                                                │ Alert!   │    │ │  │
│  │  │                                                │ via      │    │ │  │
│  │  │                                                │ Sidekick │    │ │  │
│  │  │                                                └──────────┘    │ │  │
│  │  └──────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                       │  │
│  │  ┌──────────────────┐  ┌──────────────────────────────────────────┐  │  │
│  │  │ Falco Agent      │  │ Falcosidekick                            │  │  │
│  │  │ (DaemonSet)      │  │ (Deployment)                             │  │  │
│  │  │                  │  │                                          │  │  │
│  │  │ 1 pod per node   │  │ Receives alerts from Falco              │  │  │
│  │  │ eBPF driver      │  │ Forwards to:                            │  │  │
│  │  │ (auto-detected)  │  │   - Slack                               │  │  │
│  │  │                  │  │   - Alertmanager                        │  │  │
│  │  │ Monitors:        │  │   - Loki (logs)                         │  │  │
│  │  │  - syscalls      │  │   - Webhook                             │  │  │
│  │  │  - file access   │  │                                          │  │  │
│  │  │  - network       │  │                                          │  │  │
│  │  │  - process exec  │  │                                          │  │  │
│  │  └──────────────────┘  └──────────────────────────────────────────┘  │  │
│  │                                                                       │  │
│  │  Resources: 100m-500m CPU, 256Mi-512Mi RAM (per node)                │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Kyverno — Admission Policies

### Planned Policies for Zenith

```
┌─────────────────────────────────────────────────────────────────────────┐
│              KYVERNO POLICIES (planned/implemented)                       │
│                                                                          │
│  VALIDATE POLICIES (block bad resources):                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  1. require-image-from-harbor                                      │ │
│  │     Scope: customer namespaces (zenith-apps, zenith-builds)        │ │
│  │     Rule: Images must come from registry.stage.freezenith.com      │ │
│  │     Why: Prevent pulling malicious images from Docker Hub          │ │
│  │                                                                    │ │
│  │  2. require-resource-limits                                        │ │
│  │     Scope: all namespaces except kube-system                       │ │
│  │     Rule: All containers must have CPU + memory limits set         │ │
│  │     Why: Prevent resource exhaustion from unbounded containers     │ │
│  │                                                                    │ │
│  │  3. require-labels                                                 │ │
│  │     Scope: all namespaces                                          │ │
│  │     Rule: Pods must have app.kubernetes.io/name label              │ │
│  │     Why: Observability — every pod must be identifiable            │ │
│  │                                                                    │ │
│  │  4. block-privileged-containers                                    │ │
│  │     Scope: customer namespaces                                     │ │
│  │     Rule: securityContext.privileged must be false                  │ │
│  │     Why: Prevent container escape attacks                          │ │
│  │                                                                    │ │
│  │  5. block-host-network                                             │ │
│  │     Scope: customer namespaces                                     │ │
│  │     Rule: hostNetwork must be false                                │ │
│  │     Why: Prevent access to node-level network                     │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  MUTATE POLICIES (auto-fix resources):                                   │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  6. add-default-security-context                                   │ │
│  │     Scope: customer namespaces                                     │ │
│  │     Action: If no securityContext → add:                           │ │
│  │       runAsNonRoot: true                                           │ │
│  │       readOnlyRootFilesystem: true                                │ │
│  │       allowPrivilegeEscalation: false                             │ │
│  │                                                                    │ │
│  │  7. add-image-pull-secret                                          │ │
│  │     Scope: customer namespaces                                     │ │
│  │     Action: Auto-add imagePullSecrets: [harbor-registry]          │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  GENERATE POLICIES (auto-create resources):                              │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  8. generate-network-policy                                        │ │
│  │     Trigger: new namespace with label tier=customer                │ │
│  │     Action: Auto-create CiliumNetworkPolicy (default deny +       │ │
│  │       allow: Traefik, APISIX, Prometheus, CoreDNS, free-pg)       │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. How Kyverno Blocks a Bad Deployment

```
Developer: kubectl apply -f deployment.yaml
  (image: docker.io/evil/crypto-miner:latest, no resource limits)
    │
    ▼
┌──────────────────────────────────────────────────────────────────────┐
│  K8s API Server                                                       │
│                                                                       │
│  1. Receives CREATE Deployment request                                │
│  2. Calls Kyverno admission webhook (before persisting)               │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ KYVERNO WEBHOOK                                                │   │
│  │                                                                │   │
│  │ Check policy: require-image-from-harbor                        │   │
│  │   Image: docker.io/evil/crypto-miner:latest                    │   │
│  │   Expected: registry.stage.freezenith.com/*                    │   │
│  │   Result: ✗ VIOLATION                                          │   │
│  │                                                                │   │
│  │ Check policy: require-resource-limits                          │   │
│  │   limits.cpu: not set                                          │   │
│  │   limits.memory: not set                                       │   │
│  │   Result: ✗ VIOLATION                                          │   │
│  │                                                                │   │
│  │ DECISION: DENY                                                 │   │
│  │ Message: "Image must be from registry.stage.freezenith.com.    │   │
│  │           Container must have resource limits."                 │   │
│  └────────────────────────────────────────────────────────────────┘   │
│                                                                       │
│  3. Returns 403 Forbidden to kubectl                                  │
│  4. Deployment is NOT created                                         │
└──────────────────────────────────────────────────────────────────────┘

kubectl output:
  Error from server: admission webhook "validate.kyverno.svc" denied
  the request: [require-image-from-harbor] Image docker.io/evil/
  crypto-miner:latest is not from allowed registry.
```

---

## 6. Falco — Runtime Security

### What Falco Detects

```
┌─────────────────────────────────────────────────────────────────────────┐
│              FALCO DETECTION RULES (built-in + custom)                    │
│                                                                          │
│  HIGH SEVERITY:                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Shell spawned in container                                        │ │
│  │    Rule: process exec = /bin/bash, /bin/sh in non-shell container │ │
│  │    Alert: "Terminal shell in container"                            │ │
│  │    Why: Attacker got remote code execution                        │ │
│  │                                                                    │ │
│  │  Sensitive file read                                               │ │
│  │    Rule: open(/etc/shadow, /etc/passwd, /proc/*/environ)          │ │
│  │    Alert: "Sensitive file opened for reading"                     │ │
│  │    Why: Credential theft attempt                                  │ │
│  │                                                                    │ │
│  │  Container escape attempt                                          │ │
│  │    Rule: mount syscall + nsenter + /proc/1/ns                     │ │
│  │    Alert: "Possible container escape"                             │ │
│  │    Why: Attacker trying to reach host                             │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  MEDIUM SEVERITY:                                                        │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  Unexpected network connection                                     │ │
│  │    Rule: connect() to IP not in allowlist                         │ │
│  │    Alert: "Outbound connection to unusual IP"                     │ │
│  │    Why: Possible data exfiltration or C2 communication            │ │
│  │                                                                    │ │
│  │  Package manager in container                                      │ │
│  │    Rule: exec of apt/yum/pip/npm in running container             │ │
│  │    Alert: "Package management in container"                       │ │
│  │    Why: Containers should be immutable — installing packages      │ │
│  │         means something is wrong                                  │ │
│  │                                                                    │ │
│  │  Crypto mining indicators                                          │ │
│  │    Rule: process name matches known miners + high CPU             │ │
│  │    Alert: "Possible crypto mining"                                │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 7. How Falco Detects an Attack

```
Attacker exploits vulnerability in customer app
    │
    │  RCE: executes "cat /etc/shadow" inside container
    ▼
┌──────────────────────────────────────────────────────────────────────┐
│  LINUX KERNEL                                                         │
│                                                                       │
│  1. Container process calls open("/etc/shadow", O_RDONLY)            │
│  2. Kernel executes the syscall                                       │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐   │
│  │ FALCO eBPF PROGRAM (attached to syscall hook)                  │   │
│  │                                                                │   │
│  │ 1. Captures syscall event:                                     │   │
│  │    process: cat                                                │   │
│  │    file: /etc/shadow                                           │   │
│  │    operation: open (read)                                      │   │
│  │    container: customer-app-xyz                                 │   │
│  │    namespace: zenith-apps                                      │   │
│  │    pod: customer-app-xyz-abc123                                │   │
│  │                                                                │   │
│  │ 2. Match against rules:                                        │   │
│  │    Rule: "Read sensitive file untrusted"                       │   │
│  │    Match: ✓ /etc/shadow is in sensitive file list              │   │
│  │    Severity: WARNING                                           │   │
│  │                                                                │   │
│  │ 3. Generate alert event                                        │   │
│  └────────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
           │
           │ Alert event
           ▼
┌──────────────────────────────────────────────────────────────────────┐
│  FALCOSIDEKICK                                                        │
│                                                                       │
│  Receives alert and forwards to:                                      │
│                                                                       │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐                   │
│  │ Slack       │  │ Alertmgr   │  │ Loki (logs)  │                   │
│  │ #security   │  │ (paging)   │  │ (searchable) │                   │
│  │ channel     │  │            │  │              │                   │
│  └────────────┘  └────────────┘  └──────────────┘                   │
│                                                                       │
│  Alert message:                                                       │
│  "WARNING: Sensitive file opened for reading                          │
│   (user=root, command=cat /etc/shadow,                                │
│    container=customer-app-xyz,                                        │
│    namespace=zenith-apps,                                             │
│    pod=customer-app-xyz-abc123)"                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration Reference

### Kyverno

**File:** `infra/terraform/modules/k8s-platform/security.tf`

| Setting | Value |
|---------|-------|
| Namespace | kyverno |
| Replicas | 1 |
| Resources | 100m-500m CPU, 256Mi-512Mi RAM |

### Falco

**File:** `infra/terraform/modules/k8s-platform/security.tf`

| Setting | Value |
|---------|-------|
| Namespace | falco |
| Type | DaemonSet (1 per node) |
| Driver | auto (eBPF preferred, falls back to kernel module) |
| Falcosidekick | enabled |
| Resources | 100m-500m CPU, 256Mi-512Mi RAM (per node) |

---

## 9. Troubleshooting

### Deployment rejected by Kyverno

```bash
# 1. See which policy rejected it
kubectl get events -n <namespace> --sort-by=.metadata.creationTimestamp | grep kyverno

# 2. Check Kyverno policy reports
kubectl get policyreport -A
kubectl get clusterpolicyreport

# 3. Check Kyverno logs
kubectl logs -n kyverno deploy/kyverno --tail=50

# 4. Temporarily exclude a namespace (emergency only):
# Add label to namespace: policies.kyverno.io/exclude: "true"
```

### Falco not generating alerts

```bash
# 1. Check Falco pods
kubectl get pods -n falco -o wide

# 2. Check Falco logs
kubectl logs -n falco ds/falco --tail=50

# 3. Check driver status (eBPF vs kernel module)
kubectl logs -n falco ds/falco | grep driver

# 4. Generate a test alert
kubectl exec -it <any-pod> -- cat /etc/shadow
# Should produce: "WARNING: Sensitive file opened for reading"

# 5. Check Falcosidekick
kubectl logs -n falco deploy/falco-falcosidekick --tail=50
```

---

## 10. Upgrade Path

### Upgrading Kyverno

```bash
terraform plan -target=helm_release.kyverno
terraform apply -target=helm_release.kyverno
# Check: kubectl get pods -n kyverno
# Verify policies still active: kubectl get clusterpolicy
```

### Upgrading Falco

```bash
terraform plan -target=helm_release.falco
terraform apply -target=helm_release.falco
# IMPORTANT: Falco driver may need recompilation after kernel upgrade
# Check: kubectl logs -n falco ds/falco | grep "driver loaded"
```

### Adding a new Kyverno policy

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: my-new-policy
spec:
  validationFailureAction: Enforce   # Enforce = block, Audit = log only
  rules:
    - name: my-rule
      match:
        resources:
          kinds: ["Pod"]
          namespaces: ["zenith-apps"]
      validate:
        message: "Custom message shown when blocked"
        pattern:
          spec:
            containers:
              - resources:
                  limits:
                    memory: "?*"   # Must be set
```
