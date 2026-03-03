# 14 — Cilium Networking & Security

> **Purpose:** Understand how pod networking, encryption, network policies, and flow observability work in Zenith.
> **Audience:** Any developer who needs to debug connectivity, create network policies, or understand traffic flows.
> **Last Updated:** 2026-03-03
> **Related:** [02-phase2-ansible-k3s.md](./02-phase2-ansible-k3s.md) (Cilium installation via Ansible), [06-security-model.md](./06-security-model.md) (security threat model), [08-observability.md](./08-observability.md) (Hubble dashboards)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Why We Chose It](#2-why-we-chose-it)
3. [Architecture Diagram](#3-architecture-diagram)
4. [How Pod-to-Pod Communication Works](#4-how-pod-to-pod-communication-works)
5. [WireGuard Encryption](#5-wireguard-encryption)
6. [Network Policies](#6-network-policies)
7. [Hubble Observability](#7-hubble-observability)
8. [Configuration Reference](#8-configuration-reference)
9. [Troubleshooting](#9-troubleshooting)
10. [Upgrade Path](#10-upgrade-path)

---

## 1. Overview

**Cilium** replaces the default Flannel CNI in k3s. It provides:

- **eBPF-based networking** — Pod-to-pod communication without iptables overhead
- **kube-proxy replacement** — Service routing in eBPF (faster than iptables)
- **WireGuard encryption** — All pod-to-pod traffic is encrypted transparently
- **CiliumNetworkPolicy** — L3/L4/L7 network policies (more powerful than standard K8s NetworkPolicy)
- **Hubble** — Real-time network flow observability (who talks to whom, DNS, HTTP, TCP)

```
What Cilium does for you (invisible to application code):

  Pod A ──────── WireGuard encrypted tunnel ──────── Pod B
           (automatic, zero config in app code)
```

---

## 2. Why We Chose It

| Feature | Cilium | Flannel | Calico | Weave |
|---------|--------|---------|--------|-------|
| eBPF-based | Yes | No (iptables) | Partial | No |
| kube-proxy replacement | Yes | No | No | No |
| WireGuard encryption | Built-in | No | Add-on | Yes |
| L7 network policies | Yes (HTTP, gRPC, DNS) | No | No | No |
| Flow observability | Hubble (built-in) | No | No | No |
| Service mesh (optional) | Yes | No | No | No |
| Performance | Best (kernel-bypass) | Good | Good | Poor |

**Decision:** Cilium provides encryption, observability, and L7 policies in one tool. With Flannel, we'd need 3 separate tools for the same capabilities.

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         CILIUM IN THE ZENITH CLUSTER                        │
│                         Namespace: kube-system                              │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    CILIUM AGENT (DaemonSet — 1 per node)              │  │
│  │                                                                       │  │
│  │  ┌─────────────────────────────────────────────────────────────────┐  │  │
│  │  │                    eBPF DATAPATH (in Linux kernel)               │  │  │
│  │  │                                                                 │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │  │  │
│  │  │  │ Pod      │  │ Service  │  │ Network  │  │ WireGuard    │   │  │  │
│  │  │  │ routing  │  │ load     │  │ policy   │  │ encryption   │   │  │  │
│  │  │  │ (L3/L4)  │  │ balance  │  │ enforce  │  │ (transparent)│   │  │  │
│  │  │  │          │  │ (kube-   │  │ (L3-L7)  │  │              │   │  │  │
│  │  │  │ No       │  │  proxy   │  │          │  │ All pod-to-  │   │  │  │
│  │  │  │ iptables │  │  replace)│  │ Drop     │  │ pod traffic  │   │  │  │
│  │  │  │ needed   │  │          │  │ denied   │  │ encrypted    │   │  │  │
│  │  │  │          │  │          │  │ traffic  │  │              │   │  │  │
│  │  │  └──────────┘  └──────────┘  └──────────┘  └──────────────┘   │  │  │
│  │  └─────────────────────────────────────────────────────────────────┘  │  │
│  │                                                                       │  │
│  │  Monitors:                                                            │  │
│  │    - All pod network interfaces (veth pairs)                          │  │
│  │    - All Service ClusterIPs                                           │  │
│  │    - All CiliumNetworkPolicy CRDs                                     │  │
│  │    - DNS queries (for FQDN-based policies)                            │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    CILIUM OPERATOR (Deployment)                        │  │
│  │                                                                       │  │
│  │  Manages:                                                             │  │
│  │    - IP address allocation (CiliumNode IPAM)                          │  │
│  │    - CiliumIdentity garbage collection                                │  │
│  │    - CiliumEndpoint lifecycle                                         │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                    HUBBLE (Observability)                              │  │
│  │                                                                       │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌────────────────────────────┐  │  │
│  │  │ Hubble Relay │  │ Hubble UI    │  │ Hubble Metrics             │  │  │
│  │  │ (Deployment) │  │ (Deployment) │  │ (exported to Prometheus)   │  │  │
│  │  │              │  │              │  │                            │  │  │
│  │  │ Aggregates   │  │ Web UI at    │  │ Counters: flows, drops,   │  │  │
│  │  │ flow events  │  │ hubble.stage │  │ DNS, HTTP, policy verdicts│  │  │
│  │  │ from all     │  │ .freezenith  │  │                            │  │  │
│  │  │ agents       │  │ .com         │  │ Scraped by Prometheus      │  │  │
│  │  └──────────────┘  └──────────────┘  └────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. How Pod-to-Pod Communication Works

```
┌─────────────────────────────────────────────────────────────────────────┐
│              POD-TO-POD COMMUNICATION (same node)                        │
│                                                                          │
│  zenith-api pod (10.42.0.15)                                             │
│       │                                                                  │
│       │ 1. App sends TCP packet to free-pg-rw:5432                      │
│       │    (Destination: Service ClusterIP 10.43.12.5:5432)              │
│       ▼                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐│
│  │  Cilium eBPF (in kernel)                                            ││
│  │                                                                     ││
│  │  Step 1: Service resolution (kube-proxy replacement)                ││
│  │    ClusterIP 10.43.12.5 → Pod IP 10.42.0.22 (DNAT in eBPF)        ││
│  │                                                                     ││
│  │  Step 2: Network policy check                                       ││
│  │    Source: zenith-api (identity 12345)                               ││
│  │    Dest: free-pg (identity 67890)                                   ││
│  │    Port: 5432 (TCP)                                                 ││
│  │    Policy verdict: ALLOWED ✓                                        ││
│  │                                                                     ││
│  │  Step 3: WireGuard encryption                                       ││
│  │    Encrypt packet with WireGuard key                                ││
│  │    (even on same node — defense in depth)                           ││
│  │                                                                     ││
│  │  Step 4: Deliver to destination pod's veth interface                ││
│  │    Decrypt at destination pod's eBPF hook                           ││
│  │    Deliver to free-pg pod                                           ││
│  └─────────────────────────────────────────────────────────────────────┘│
│       │                                                                  │
│       ▼                                                                  │
│  free-pg pod (10.42.0.22:5432)                                          │
│  PostgreSQL receives the connection                                      │
└─────────────────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────┐
│              POD-TO-POD COMMUNICATION (cross-node, production)           │
│                                                                          │
│  Node 1                                    Node 2                        │
│  ┌──────────────────────┐                 ┌──────────────────────┐      │
│  │ zenith-api pod        │                 │ free-pg pod           │      │
│  │ 10.42.0.15            │                 │ 10.42.1.22            │      │
│  └──────────┬────────────┘                 └──────────▲────────────┘      │
│             │                                         │                  │
│             ▼                                         │                  │
│  ┌──────────────────────┐                 ┌──────────┴────────────┐      │
│  │ Cilium eBPF          │                 │ Cilium eBPF           │      │
│  │ 1. Service resolution│                 │ 4. Decrypt WireGuard  │      │
│  │ 2. Policy check      │                 │ 5. Policy check       │      │
│  │ 3. WireGuard encrypt │                 │ 6. Deliver to pod     │      │
│  └──────────┬────────────┘                 └──────────▲────────────┘      │
│             │                                         │                  │
│             ▼                                         │                  │
│  ┌──────────────────────────────────────────────────────────────────┐    │
│  │              WIREGUARD TUNNEL (encrypted, between nodes)         │    │
│  │              Uses node's WireGuard keys (auto-rotated)           │    │
│  │              Protocol: UDP :51871                                 │    │
│  └──────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. WireGuard Encryption

```
┌─────────────────────────────────────────────────────────────────────────┐
│              WIREGUARD ENCRYPTION IN CILIUM                              │
│                                                                          │
│  What is encrypted:                                                      │
│    ✓ ALL pod-to-pod traffic (same node and cross-node)                  │
│    ✓ Service traffic (ClusterIP, NodePort)                              │
│    ✗ Traffic to external services (S3, Cloudflare, etc.)                │
│    ✗ Kubelet-to-API-server (uses TLS separately)                        │
│                                                                          │
│  How it works:                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │ 1. Cilium agent generates WireGuard keypair on each node           │ │
│  │ 2. Keys are exchanged via CiliumNode CRDs (stored in K8s API)     │ │
│  │ 3. Each node creates a WireGuard interface (cilium_wg0)            │ │
│  │ 4. eBPF programs route pod traffic through WireGuard interface     │ │
│  │ 5. WireGuard encrypts at kernel level (very fast, ~1Gbps+)        │ │
│  │ 6. Keys are auto-rotated periodically (no manual intervention)     │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Configuration (from cilium/tasks/main.yml):                             │
│    --set encryption.enabled=true                                         │
│    --set encryption.type=wireguard                                       │
│                                                                          │
│  Why WireGuard over IPsec:                                               │
│    - Simpler (4000 lines of code vs 400,000 for IPsec)                  │
│    - Faster (modern crypto: ChaCha20-Poly1305)                          │
│    - Smaller attack surface                                              │
│    - No certificate management                                           │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Network Policies

### Default Deny for Customer Namespaces

```
┌─────────────────────────────────────────────────────────────────────────┐
│              NETWORK POLICY STRATEGY                                     │
│                                                                          │
│  PLATFORM NAMESPACES (kube-system, apisix, monitoring, etc.):           │
│    Default: Allow all (infrastructure needs to communicate freely)        │
│                                                                          │
│  CUSTOMER NAMESPACES (zenith-apps, zenith-builds):                      │
│    Default: DENY all ingress + egress                                    │
│    Explicit allows:                                                      │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  CiliumNetworkPolicy for customer namespace                        │ │
│  │                                                                    │ │
│  │  ALLOW INGRESS:                                                    │ │
│  │    ✓ Traefik → customer pods (port 3000)   ← serve web traffic    │ │
│  │    ✓ APISIX → customer pods (port 8080)    ← serve API traffic    │ │
│  │    ✓ Prometheus → customer pods (/metrics) ← scrape metrics       │ │
│  │                                                                    │ │
│  │  ALLOW EGRESS:                                                     │ │
│  │    ✓ customer → free-pg (port 5432)        ← database access      │ │
│  │    ✓ customer → CoreDNS (port 53)          ← DNS resolution       │ │
│  │    ✓ customer → external (port 443)        ← external APIs        │ │
│  │    ✗ customer → other customers            ← BLOCKED (isolation)  │ │
│  │    ✗ customer → kube-system                ← BLOCKED (security)   │ │
│  │    ✗ customer → keycloak                   ← BLOCKED (internal)   │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  TENANT ISOLATION:                                                       │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  Customer A pods ──✗──▶ Customer B pods  (BLOCKED by policy)      │ │
│  │  Customer A pods ──✓──▶ free-pg          (ALLOWED — own DB only)  │ │
│  │  Customer A pods ──✓──▶ Hetzner S3       (ALLOWED — own bucket)   │ │
│  │  Customer A pods ──✗──▶ Keycloak admin   (BLOCKED — API only)     │ │
│  │                                                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

### L7 Policy Example (HTTP-aware)

```yaml
# Cilium can filter at HTTP level (not just TCP port)
apiVersion: cilium.io/v2
kind: CiliumNetworkPolicy
metadata:
  name: customer-db-access
  namespace: zenith-apps
spec:
  endpointSelector:
    matchLabels:
      app: customer-backend
  egress:
    - toEndpoints:
        - matchLabels:
            cnpg.io/cluster: free-pg
      toPorts:
        - ports:
            - port: "5432"
              protocol: TCP
    - toEndpoints:
        - matchLabels:
            k8s:io.kubernetes.pod.namespace: kube-system
            k8s-app: kube-dns
      toPorts:
        - ports:
            - port: "53"
              protocol: UDP
```

---

## 7. Hubble Observability

```
┌─────────────────────────────────────────────────────────────────────────┐
│              HUBBLE — NETWORK FLOW OBSERVABILITY                         │
│                                                                          │
│  What Hubble shows you:                                                  │
│    - Every network connection between pods (real-time)                   │
│    - DNS queries and responses                                           │
│    - HTTP request/response details (method, path, status)                │
│    - TCP connection state (SYN, ACK, FIN, RST)                          │
│    - Network policy verdicts (ALLOWED / DROPPED)                         │
│    - Packet drop reasons                                                 │
│                                                                          │
│  Architecture:                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                                                                    │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                        │ │
│  │  │ Cilium   │  │ Cilium   │  │ Cilium   │  (one per node)        │ │
│  │  │ Agent    │  │ Agent    │  │ Agent    │                         │ │
│  │  │ + Hubble │  │ + Hubble │  │ + Hubble │                         │ │
│  │  │ Observer │  │ Observer │  │ Observer │                         │ │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘                        │ │
│  │       │              │              │                              │ │
│  │       └──────────────┼──────────────┘                              │ │
│  │                      │ gRPC                                       │ │
│  │                      ▼                                             │ │
│  │              ┌──────────────┐                                      │ │
│  │              │ Hubble Relay │  Aggregates flows from all nodes     │ │
│  │              │ (Deployment) │                                      │ │
│  │              └──────┬───────┘                                      │ │
│  │                     │                                              │ │
│  │          ┌──────────┴──────────┐                                   │ │
│  │          │                     │                                   │ │
│  │          ▼                     ▼                                   │ │
│  │  ┌──────────────┐    ┌──────────────────┐                         │ │
│  │  │ Hubble UI    │    │ Hubble Metrics   │                         │ │
│  │  │ Web interface│    │ → Prometheus     │                         │ │
│  │  │ hubble.stage │    │ → Grafana        │                         │ │
│  │  │ .freezenith  │    │ dashboards       │                         │ │
│  │  │ .com         │    │                  │                         │ │
│  │  └──────────────┘    └──────────────────┘                         │ │
│  └────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│  Access Hubble UI:                                                       │
│    https://hubble.stage.freezenith.com                                   │
│    (via Traefik IngressRoute in kube-system)                             │
│                                                                          │
│  Access Hubble CLI:                                                      │
│    hubble observe --namespace zenith-staging                             │
│    hubble observe --verdict DROPPED                                      │
│    hubble observe --to-pod zenith-api                                    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Configuration Reference

### Installation (Ansible)

**File:** `infra/ansible/roles/cilium/tasks/main.yml`

```bash
cilium install \
  --set kubeProxyReplacement=true \
  --set encryption.enabled=true \
  --set encryption.type=wireguard \
  --set hubble.enabled=true \
  --set hubble.ui.enabled=true
```

### k3s Flags (Ansible)

**File:** `infra/ansible/roles/k3s/tasks/main.yml`

```bash
# k3s is installed WITHOUT Flannel (Cilium replaces it):
--flannel-backend=none      # Disable Flannel CNI
--disable-network-policy    # Disable k3s network policy (Cilium handles it)
--disable=servicelb         # Disable k3s ServiceLB (not needed)
```

### Hubble UI IngressRoute

**File:** `infra/terraform/modules/k8s-platform/observability.tf`

```
IngressRoute: hubble-ui
Namespace: kube-system
Host: hubble.stage.freezenith.com
Backend: hubble-ui:80
TLS: hubble-tls
```

---

## 9. Troubleshooting

### Pod can't reach another pod

```bash
# 1. Check Cilium status
cilium status
# All components should be "OK"

# 2. Check if endpoints are discovered
cilium endpoint list
# Both source and destination pods should appear

# 3. Check policy verdict
hubble observe --from-pod <namespace>/<source-pod> --to-pod <namespace>/<dest-pod>
# Look for DROPPED verdicts

# 4. Check network policy
kubectl get ciliumnetworkpolicy -n <namespace>
kubectl describe ciliumnetworkpolicy <name> -n <namespace>

# 5. Test connectivity
cilium connectivity test
# Runs a comprehensive connectivity test suite
```

### DNS not resolving inside pods

```bash
# 1. Check CoreDNS
kubectl get pods -n kube-system -l k8s-app=kube-dns

# 2. Check Hubble for DNS flows
hubble observe --protocol DNS --namespace <namespace>

# 3. Check if DNS egress is allowed by policy
kubectl get ciliumnetworkpolicy -n <namespace> -o yaml | grep -A5 "port: .53"

# 4. Test from inside pod
kubectl exec -it <pod> -- nslookup kubernetes.default.svc.cluster.local
```

### WireGuard not working

```bash
# 1. Check WireGuard interface
kubectl exec -n kube-system ds/cilium -- cilium encrypt status
# Should show: Encryption: WireGuard

# 2. Check WireGuard peers
kubectl exec -n kube-system ds/cilium -- wg show

# 3. Check node-to-node connectivity
# Port 51871 (UDP) must be open between nodes
```

### Hubble UI not loading

```bash
# 1. Check Hubble pods
kubectl get pods -n kube-system -l k8s-app=hubble-ui
kubectl get pods -n kube-system -l k8s-app=hubble-relay

# 2. Check IngressRoute
kubectl get ingressroute hubble-ui -n kube-system -o yaml

# 3. Check cert
kubectl get certificate -n kube-system | grep hubble
```

---

## 10. Upgrade Path

### Upgrading Cilium

```bash
# SSH to the VM
ssh root@zen-stage

# Upgrade Cilium CLI
CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
curl -L --fail --remote-name-all \
  https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-amd64.tar.gz
sudo tar xzvfC cilium-linux-amd64.tar.gz /usr/local/bin

# Upgrade Cilium in-cluster
cilium upgrade --version v1.17.0

# Verify
cilium status
cilium connectivity test
```

### Adding a new network policy

```bash
# 1. Write CiliumNetworkPolicy YAML
# 2. Apply: kubectl apply -f policy.yaml
# 3. Verify: hubble observe --namespace <ns> --verdict DROPPED
# 4. Test: curl from source pod to destination
```
