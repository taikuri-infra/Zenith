# Galaxy - Complete Architecture

> Operator-driven PaaS on Hetzner. Everything is a CRD. Everything maps to Hetzner.

---

## Core Principle

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  User sees: Simple UI (like Supabase/Fly.io)                    │
│  Backend does: Creates a CRD in Kubernetes                       │
│  Galaxy Operator does: Creates Hetzner resources + service CRDs  │
│  Service Operators do: Create the actual services                │
│                                                                  │
│  User NEVER sees: K8s, operators, PVCs, ingress, nodes          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## System Flow

```
┌─────────┐    ┌─────────────┐    ┌────────────────┐    ┌──────────────────┐
│         │    │             │    │                │    │                  │
│  Web UI │───▶│  Backend    │───▶│  Kubernetes    │───▶│  Galaxy Operator │
│ (Next)  │    │  (Go API)   │    │  API Server    │    │  (watches CRDs)  │
│         │    │             │    │  (CRD created) │    │                  │
└─────────┘    └─────────────┘    └────────────────┘    └────────┬─────────┘
                                                                 │
                                                    ┌────────────┼────────────┐
                                                    │            │            │
                                                    ▼            ▼            ▼
                                            ┌──────────┐  ┌──────────┐  ┌──────────┐
                                            │ Hetzner  │  │ Service  │  │ Service  │
                                            │ API      │  │ CRD      │  │ CRD      │
                                            │          │  │ created  │  │ created  │
                                            │ Volume   │  │          │  │          │
                                            │ Network  │  │ CNPG     │  │ Redis    │
                                            │ LB       │  │ Operator │  │ Operator │
                                            │ Firewall │  │ picks up │  │ picks up │
                                            │ DNS      │  │          │  │          │
                                            └──────────┘  └──────────┘  └──────────┘
```

### Concrete Example: User clicks "Add PostgreSQL 16, 20GB"

```
Step 1: Frontend
  └── User clicks "Add Database" → selects PostgreSQL → 20GB → Create

Step 2: Backend (Go API)
  └── POST /api/v1/projects/{id}/databases
      Body: { engine: "postgresql", version: "16", storage: "20Gi" }
  └── Backend authenticates user, validates request
  └── Backend creates CRD in Kubernetes:

      apiVersion: galaxy.dev/v1alpha1
      kind: Database
      metadata:
        name: db-a1b2c3
        namespace: galaxy-project-xyz
      spec:
        engine: postgresql
        version: "16"
        storage: 20Gi

Step 3: Galaxy Operator (watching Database CRDs)
  └── Sees new Database CRD
  └── Calls Hetzner API: create Volume (20GB, fsn1)
  └── Hetzner returns: volume_id: 12345678
  └── Creates PersistentVolume:
        spec:
          csi:
            driver: csi.hetzner.cloud
            volumeHandle: "12345678"
          capacity:
            storage: 20Gi
  └── Creates PersistentVolumeClaim (bound to PV)
  └── Creates CloudNativePG Cluster CR:
        apiVersion: postgresql.cnpg.io/v1
        kind: Cluster
        metadata:
          name: db-a1b2c3
        spec:
          instances: 1
          storage:
            pvcTemplate:
              resources:
                requests:
                  storage: 20Gi
              # references the PVC Galaxy created

Step 4: CloudNativePG Operator (watching Cluster CRDs)
  └── Sees new Cluster CR
  └── Creates PostgreSQL pod with the PVC
  └── Creates Service (ClusterIP)
  └── Generates credentials
  └── Updates Cluster status: ready

Step 5: Galaxy Operator (watching status)
  └── Sees CloudNativePG Cluster is ready
  └── Reads connection credentials
  └── Creates K8s Secret with connection string
  └── Updates Galaxy Database CRD status:
        status:
          phase: Ready
          connectionString: postgres://user:pass@db-a1b2c3:5432/app
          hetznerVolumeId: "12345678"

Step 6: Frontend
  └── Polls API → sees status: Ready
  └── Shows: "PostgreSQL Ready ✓"
  └── Shows: Connection string (copy button)
```

---

## Complete Service Map

### Data Services

| User Sees | Galaxy CRD | Galaxy Operator Creates | Service Operator | Hetzner Resource |
|-----------|-----------|------------------------|-----------------|-----------------|
| PostgreSQL | `Database` (engine: postgresql) | PV + PVC + CNPG Cluster CR | CloudNativePG | Volume |
| MySQL | `Database` (engine: mysql) | PV + PVC + MySQL CR | MySQL Operator | Volume |
| MongoDB | `Database` (engine: mongodb) | PV + PVC + MongoDB CR | MongoDB Community Op | Volume |
| Redis | `Database` (engine: redis) | PV + PVC + Redis CR | Redis Operator | Volume |
| S3 Storage | `ObjectStore` | Calls Hetzner S3 API | (none - native Hetzner) | Object Storage |
| KV Store | `KeyValueStore` | PV + PVC + NATS CR | NATS Operator | Volume |
| Backup | `BackupPolicy` | CronJob (pg_dump/mongodump → S3) | (built-in) | Object Storage |

### Compute Services

| User Sees | Galaxy CRD | Galaxy Operator Creates | Service Operator | Hetzner Resource |
|-----------|-----------|------------------------|-----------------|-----------------|
| Deploy App | `Application` | Deployment + Service + Ingress | (built-in) | (runs on nodes) |
| Container Registry | `Registry` | Harbor/Nexus Helm release + PVC | Harbor Operator | Volume |
| Build from GitHub | `Build` | Kaniko Job | (built-in) | (runs on nodes) |
| Add Node (Planet) | `Planet` | VM via Hetzner API + k3s join | (built-in) | Cloud Server |
| Cron Job | `CronTask` | K8s CronJob | (built-in) | (runs on nodes) |

### Networking Services

| User Sees | Galaxy CRD | Galaxy Operator Creates | Hetzner Resource |
|-----------|-----------|------------------------|-----------------|
| Custom Domain | `Domain` | Ingress + cert-manager Certificate | (none) |
| Load Balancer | `LoadBalancer` | K8s Service type:LB | Hetzner LB |
| Firewall | `Firewall` | Calls Hetzner Firewall API | Hetzner Firewall |
| Private Network | `Network` | Calls Hetzner Network API | Hetzner Network |
| Floating IP | `FloatingIP` | Calls Hetzner Floating IP API | Hetzner Floating IP |
| DNS | `DNSZone` / `DNSRecord` | Calls Hetzner DNS API | Hetzner DNS |
| VPN Peering | `VPNPeer` | WireGuard pod + config | (runs on nodes) |
| API Gateway | `Gateway` | Traefik/Kong IngressRoute | (runs on nodes) |

### Hybrid Cloud Services

| User Sees | Galaxy CRD | Galaxy Operator Creates | Hetzner Resource |
|-----------|-----------|------------------------|-----------------|
| Cloud Connection | `CloudConnector` | StrongSwan/WireGuard pod + routes + NetworkPolicy | (runs on nodes) |

**CloudConnector** enables encrypted tunnels to external clouds (AWS, GCP, Azure) or on-prem datacenters.
This makes Zenith a **hybrid cloud platform** - apps on Zenith can reach services in AWS VPC, GCP VPC, Azure VNet, or any IPsec-capable network.

**How it works:**
```
Zenith Cluster (Hetzner)              External Cloud (AWS/GCP/Azure/On-Prem)
┌────────────────────────┐            ┌────────────────────────┐
│                        │            │                        │
│  ┌──────────────────┐  │   IPsec    │  ┌──────────────────┐  │
│  │ CloudConnector   │◄─┼────────────┼─►│ VPN Gateway      │  │
│  │ Pod (StrongSwan  │  │  encrypted │  │ (AWS VGW /       │  │
│  │ or WireGuard)    │  │  tunnel    │  │  GCP Cloud VPN / │  │
│  └────────┬─────────┘  │            │  │  Azure VPN GW)   │  │
│           │             │            │  └──────────────────┘  │
│  Route: 10.100.0.0/16  │            │                        │
│  → via CloudConnector   │            │  VPC: 10.100.0.0/16   │
│           │             │            │  ├── RDS (10.100.1.50)│
│  ┌────────▼─────────┐  │            │  ├── ElastiCache      │
│  │ App Pods         │  │            │  ├── Lambda            │
│  │                  │  │            │  ├── S3 (via endpoint) │
│  │ Can reach:       │  │            │  └── EC2 instances     │
│  │ 10.100.1.50:5432 │  │            │                        │
│  │ (AWS RDS)        │  │            │                        │
│  └──────────────────┘  │            │                        │
└────────────────────────┘            └────────────────────────┘
```

**Use cases:**
1. **Hybrid migration**: Customer has legacy DB on AWS RDS, wants to move apps to Zenith gradually
2. **Multi-cloud**: App on Zenith, ML pipeline on GCP, data warehouse on AWS
3. **On-prem connectivity**: Connect Zenith to customer's datacenter via IPsec
4. **Compliance**: Data stays in specific region (AWS EU) while compute runs on Hetzner

**Supported providers:**
| Provider | Connection Type | Configuration Source |
|----------|----------------|---------------------|
| AWS | Site-to-Site VPN (IPsec) | AWS VPN Gateway config download |
| GCP | Cloud VPN (IPsec) | GCP VPN tunnel config |
| Azure | VPN Gateway (IPsec) | Azure VPN config |
| On-Prem | IPsec or WireGuard | Manual config |

### Platform Services

| User Sees | Galaxy CRD | Galaxy Operator Creates | Service Operator | Hetzner Resource |
|-----------|-----------|------------------------|-----------------|-----------------|
| Auth (SSO) | `AuthRealm` | Keycloak CR + PVC | Keycloak Operator | Volume |
| Monitoring | `Monitoring` | Grafana + Prometheus stack | kube-prometheus-stack | Volume |
| Logging | `LogPipeline` | Loki + Promtail | Grafana Loki | Volume |
| Alerts | `AlertRule` | PrometheusRule CR | Prometheus Operator | (none) |

---

## All Galaxy CRDs

```
galaxy.dev/v1alpha1
│
├── Core
│   ├── Project          # namespace + isolation + billing boundary
│   ├── Application      # deploy container/code
│   ├── Build            # build from source
│   └── Planet           # add/remove nodes
│
├── Data
│   ├── Database         # PostgreSQL, MySQL, MongoDB, Redis
│   ├── ObjectStore      # S3 bucket (Hetzner Object Storage)
│   ├── KeyValueStore    # encrypted KV (NATS KV)
│   └── BackupPolicy     # scheduled backups
│
├── Networking
│   ├── Domain           # custom domain + auto SSL
│   ├── Firewall         # firewall rules (Hetzner Firewall)
│   ├── Network          # private network (Hetzner Network)
│   ├── FloatingIP       # static IP (Hetzner Floating IP)
│   ├── LoadBalancer     # external LB (Hetzner LB)
│   ├── VPNPeer          # VPN tunnel (WireGuard)
│   ├── DNSZone          # DNS zone (Hetzner DNS)
│   ├── DNSRecord        # DNS record
│   ├── Gateway          # API gateway rules
│   └── CloudConnector   # hybrid cloud tunnel (AWS/GCP/Azure/on-prem)
│
├── Platform
│   ├── Registry         # container registry (Harbor)
│   ├── AuthRealm        # auth/SSO (Keycloak)
│   ├── Monitoring       # Grafana + Prometheus
│   ├── LogPipeline      # Loki logging
│   └── AlertRule        # alerting rules
│
└── Billing (internal, not user-facing)
    ├── UsageRecord      # tracks resource usage
    └── Invoice          # billing calculation
```

---

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          GALAXY SYSTEM                                  │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      USER LAYER                                 │    │
│  │                                                                 │    │
│  │  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │    │
│  │  │  Galaxy Web   │    │  galaxyctl   │    │  kubectl     │      │    │
│  │  │  (Next.js)    │    │  (CLI)       │    │  (advanced)  │      │    │
│  │  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘      │    │
│  │         │                   │                   │               │    │
│  └─────────┼───────────────────┼───────────────────┼───────────────┘    │
│            │                   │                   │                    │
│  ┌─────────▼───────────────────▼───────────────────▼───────────────┐    │
│  │                      API LAYER                                  │    │
│  │                                                                 │    │
│  │  ┌──────────────────────────────────────────────────────┐      │    │
│  │  │  Galaxy API Server (Go)                               │      │    │
│  │  │                                                       │      │    │
│  │  │  - REST API for Web UI / CLI                          │      │    │
│  │  │  - Authentication (JWT + OAuth)                       │      │    │
│  │  │  - Authorization (RBAC)                               │      │    │
│  │  │  - Validates requests                                 │      │    │
│  │  │  - Creates/reads Galaxy CRDs in K8s                   │      │    │
│  │  │  - WebSocket for real-time logs/status                │      │    │
│  │  │                                                       │      │    │
│  │  └──────────────────────────┬───────────────────────────┘      │    │
│  │                             │                                   │    │
│  └─────────────────────────────┼───────────────────────────────────┘    │
│                                │                                        │
│                                │ creates CRDs                           │
│                                ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    OPERATOR LAYER                                │    │
│  │                                                                 │    │
│  │  ┌─────────────────────────────────────────────────────────┐   │    │
│  │  │  Galaxy Operator (Go, controller-runtime)                │   │    │
│  │  │                                                          │   │    │
│  │  │  Controllers:                                            │   │    │
│  │  │  ├── ProjectController      → namespace + RBAC + quota   │   │    │
│  │  │  ├── ApplicationController  → deployment + ingress       │   │    │
│  │  │  ├── BuildController        → kaniko jobs                │   │    │
│  │  │  ├── PlanetController       → Hetzner VM + k3s join      │   │    │
│  │  │  ├── DatabaseController     → Hetzner Volume + svc CRD   │   │    │
│  │  │  ├── ObjectStoreController  → Hetzner S3 API             │   │    │
│  │  │  ├── FirewallController     → Hetzner Firewall API       │   │    │
│  │  │  ├── NetworkController      → Hetzner Network API        │   │    │
│  │  │  ├── FloatingIPController   → Hetzner Floating IP API    │   │    │
│  │  │  ├── DNSController          → Hetzner DNS API            │   │    │
│  │  │  ├── DomainController       → Ingress + cert-manager     │   │    │
│  │  │  ├── RegistryController     → Harbor Helm release        │   │    │
│  │  │  ├── AuthRealmController    → Keycloak CR                │   │    │
│  │  │  ├── MonitoringController   → Prometheus + Grafana       │   │    │
│  │  │  ├── LogPipelineController  → Loki + Promtail            │   │    │
│  │  │  ├── GatewayController      → Traefik IngressRoute       │   │    │
│  │  │  ├── VPNPeerController      → WireGuard pod              │   │    │
│  │  │  ├── CloudConnectorCtrl    → StrongSwan/WG + routes      │   │    │
│  │  │  ├── BackupController       → CronJob → S3               │   │    │
│  │  │  └── BillingController      → UsageRecord CRDs           │   │    │
│  │  │                                                          │   │    │
│  │  │  Hetzner Client: hcloud-go (direct API, no Terraform)    │   │    │
│  │  │                                                          │   │    │
│  │  └──────────────────────────────────────────────────────────┘   │    │
│  │                                                                 │    │
│  │  Pre-installed Service Operators:                               │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐            │    │
│  │  │ CloudNativePG│ │ Redis Op     │ │ MySQL Op     │            │    │
│  │  │ (postgresql) │ │ (redis)      │ │ (mysql)      │            │    │
│  │  └──────────────┘ └──────────────┘ └──────────────┘            │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐            │    │
│  │  │ MongoDB Op   │ │ Keycloak Op  │ │ NATS Op      │            │    │
│  │  │ (mongodb)    │ │ (auth)       │ │ (kv store)   │            │    │
│  │  └──────────────┘ └──────────────┘ └──────────────┘            │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐            │    │
│  │  │ Harbor Op    │ │ Prometheus   │ │ Loki         │            │    │
│  │  │ (registry)   │ │ (monitoring) │ │ (logging)    │            │    │
│  │  └──────────────┘ └──────────────┘ └──────────────┘            │    │
│  │  ┌──────────────┐                                              │    │
│  │  │ cert-manager │                                              │    │
│  │  │ (tls)        │                                              │    │
│  │  └──────────────┘                                              │    │
│  │                                                                 │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                   INFRASTRUCTURE LAYER                          │    │
│  │                                                                 │    │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐            │    │
│  │  │ Hetzner CSI  │ │ Hetzner CCM  │ │ Traefik      │            │    │
│  │  │ Driver       │ │ (Cloud Ctrl) │ │ (Ingress)    │            │    │
│  │  │              │ │              │ │              │            │    │
│  │  │ PVC → Volume │ │ Svc → LB     │ │ Routes HTTP  │            │    │
│  │  └──────────────┘ └──────────────┘ └──────────────┘            │    │
│  │                                                                 │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        HETZNER CLOUD                                    │
│                                                                         │
│  Cloud Servers │ Volumes │ Load Balancers │ Object Storage              │
│  Networks │ Firewalls │ Floating IPs │ DNS │ StorageBox                 │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Galaxy Operator Internal Design

The Galaxy Operator is ONE binary with multiple controllers. Each controller watches one CRD type.

### Controller Pattern (all controllers follow this)

```go
// Every controller follows the same reconciliation loop:

func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the Galaxy CRD
    var db galaxyv1.Database
    if err := r.Get(ctx, req.NamespacedName, &db); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Check if being deleted
    if !db.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, &db)
    }

    // 3. Ensure Hetzner resources exist
    volume, err := r.ensureHetznerVolume(ctx, &db)
    if err != nil {
        return ctrl.Result{}, err
    }

    // 4. Ensure PV/PVC exist
    if err := r.ensurePersistentVolume(ctx, &db, volume); err != nil {
        return ctrl.Result{}, err
    }

    // 5. Ensure service operator CR exists
    if err := r.ensureServiceCR(ctx, &db); err != nil {
        return ctrl.Result{}, err
    }

    // 6. Check service status
    ready, err := r.checkServiceReady(ctx, &db)
    if err != nil {
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }

    // 7. Update Galaxy CRD status
    if ready {
        db.Status.Phase = "Ready"
        db.Status.ConnectionString = r.getConnectionString(ctx, &db)
    }
    r.Status().Update(ctx, &db)

    // 8. Create/update billing record
    r.recordUsage(ctx, &db)

    return ctrl.Result{}, nil
}
```

### CloudConnector CRD Example

```yaml
apiVersion: zenith.dev/v1alpha1
kind: CloudConnector
metadata:
  name: aws-production
  namespace: zenith-my-startup
spec:
  # Which cloud to connect to
  provider: aws          # aws | gcp | azure | custom

  # Connection type
  type: ipsec            # ipsec | wireguard

  # Remote side configuration
  remote:
    # AWS: download VPN config from AWS Console → paste values here
    gatewayIP: "52.47.xxx.xxx"          # AWS VPN endpoint
    cidr: "10.100.0.0/16"              # AWS VPC CIDR
    presharedKey:
      secretRef:
        name: aws-vpn-secret
        key: psk
    # Optional: BGP for dynamic routing
    bgp:
      enabled: false
      remoteASN: 64512

  # Local side configuration
  local:
    cidr: "10.0.0.0/16"               # Zenith cluster CIDR

  # Access control: which apps/namespaces can use this tunnel
  access:
    allowedNamespaces: ["zenith-my-startup"]
    allowedCIDRs: ["10.100.1.0/24", "10.100.2.0/24"]   # only these subnets reachable

  # Health check
  healthCheck:
    enabled: true
    remoteIP: "10.100.1.1"            # ping this IP to verify tunnel
    intervalSeconds: 30

status:
  phase: Connected        # Pending | Connecting | Connected | Failed
  tunnelIP: "10.0.200.1"
  lastHandshake: "2026-02-15T14:30:00Z"
  latency: "45ms"
```

**User's app can then access AWS services:**
```yaml
apiVersion: zenith.dev/v1alpha1
kind: Application
metadata:
  name: api-service
spec:
  env:
    # This reaches AWS RDS through the CloudConnector tunnel
    - name: LEGACY_DB_URL
      value: "postgres://admin:pass@10.100.1.50:5432/legacy_db"
    # This reaches AWS ElastiCache through the tunnel
    - name: AWS_REDIS_URL
      value: "redis://10.100.2.30:6379"
  # Declare dependency on the cloud connector
  cloudConnectors:
    - aws-production
```

### Hetzner Client (hcloud-go, NOT Terraform)

```go
// internal/provider/hetzner/client.go

type HetznerClient struct {
    client *hcloud.Client
}

func NewHetznerClient(token string) *HetznerClient {
    return &HetznerClient{
        client: hcloud.NewClient(hcloud.WithToken(token)),
    }
}

// Volume operations
func (h *HetznerClient) CreateVolume(name string, sizeGB int, location string) (*hcloud.Volume, error)
func (h *HetznerClient) DeleteVolume(id int64) error
func (h *HetznerClient) ResizeVolume(id int64, newSizeGB int) error

// Server operations (for Planets)
func (h *HetznerClient) CreateServer(name, serverType, image, location string, sshKeys []string) (*hcloud.Server, error)
func (h *HetznerClient) DeleteServer(id int64) error

// Network operations
func (h *HetznerClient) CreateNetwork(name, cidr string) (*hcloud.Network, error)
func (h *HetznerClient) DeleteNetwork(id int64) error
func (h *HetznerClient) AddSubnet(networkID int64, subnet hcloud.NetworkSubnet) error

// Firewall operations
func (h *HetznerClient) CreateFirewall(name string, rules []hcloud.FirewallRule) (*hcloud.Firewall, error)
func (h *HetznerClient) UpdateFirewall(id int64, rules []hcloud.FirewallRule) error
func (h *HetznerClient) DeleteFirewall(id int64) error

// Floating IP operations
func (h *HetznerClient) CreateFloatingIP(ipType, location, description string) (*hcloud.FloatingIP, error)
func (h *HetznerClient) AssignFloatingIP(ipID, serverID int64) error
func (h *HetznerClient) DeleteFloatingIP(id int64) error

// DNS operations (Hetzner DNS API)
func (h *HetznerClient) CreateDNSZone(name string) (*DNSZone, error)
func (h *HetznerClient) CreateDNSRecord(zoneID, recordType, name, value string) error
func (h *HetznerClient) DeleteDNSRecord(recordID string) error

// Load Balancer operations
func (h *HetznerClient) CreateLoadBalancer(name, lbType, location string) (*hcloud.LoadBalancer, error)
func (h *HetznerClient) DeleteLoadBalancer(id int64) error

// Object Storage (S3-compatible API)
func (h *HetznerClient) CreateBucket(name, region string) error
func (h *HetznerClient) DeleteBucket(name string) error
func (h *HetznerClient) GenerateS3Credentials() (accessKey, secretKey string, err error)
```

**Why hcloud-go and NOT Terraform/CDKTF:**
- Terraform has state files. State management inside an operator loop is fragile.
- hcloud-go is the official Hetzner Go SDK. Direct API calls. No intermediary.
- Faster: no terraform init/plan/apply cycle. Direct API call in milliseconds.
- Simpler: no need to manage .tfstate per resource.
- The operator IS the state manager. CRD status = state. K8s etcd = state store.

**CDKTF is used ONLY for initial cluster bootstrap** (`galaxyctl install`). After that, the operator handles everything via hcloud-go.

---

## Installation Flow (galaxyctl install)

```
$ galaxyctl install --provider hetzner --token hc_xxxxx --region fsn1

[1/8] Creating SSH key pair...                                    ✓
[2/8] Creating private network (10.0.0.0/16)...                  ✓
[3/8] Creating firewall rules...                                  ✓
[4/8] Creating control plane nodes (3x CX33)...                  ✓
      ├── galaxy-cp-1 (10.0.1.1) - master
      ├── galaxy-cp-2 (10.0.1.2) - master
      └── galaxy-cp-3 (10.0.1.3) - master
[5/8] Installing k3s cluster (HA mode)...                        ✓
[6/8] Installing infrastructure components...                     ✓
      ├── Hetzner CSI Driver
      ├── Hetzner Cloud Controller Manager
      ├── Traefik (ingress)
      ├── cert-manager
      └── Hetzner DNS webhook
[7/8] Installing Galaxy platform...                               ✓
      ├── Galaxy CRDs (20 types)
      ├── Galaxy Operator
      ├── Galaxy API Server
      ├── Galaxy Web UI
      ├── PostgreSQL (control plane DB)
      └── Service operators (CNPG, Redis, MySQL, MongoDB,
          Keycloak, Harbor, NATS, Prometheus, Loki)
[8/8] Configuring admin account...                                ✓

════════════════════════════════════════════════════════════
  Galaxy is ready!

  Dashboard:  https://console.galaxy.dev  (or your IP)
  Admin:      admin@galaxy.local
  Password:   xxxxxxxxxxxxxxxx

  Kubeconfig saved to: ~/.galaxy/kubeconfig

  Monthly infrastructure cost: ~€17/mo (3x CX33 nodes)
  Galaxy software cost: €0 (open source, Apache 2.0)
════════════════════════════════════════════════════════════
```

What `galaxyctl install` does internally:

```
1. Uses hcloud-go to:
   - Create SSH key
   - Create Network (10.0.0.0/16)
   - Create Firewall (22, 80, 443, 6443, 10250)
   - Create 3 VMs (CX33) with cloud-init

2. Cloud-init on first node:
   - Install k3s server (--cluster-init for HA)
   - curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=v1.29.2+k3s1 sh -s - server \
       --cluster-init \
       --disable=traefik \
       --tls-san=<public-ip> \
       --node-external-ip=<public-ip> \
       --flannel-iface=ens10

3. Cloud-init on nodes 2-3:
   - Join k3s cluster (--server https://node1:6443)

4. Fetch kubeconfig from node 1

5. helm install / kubectl apply:
   - hetzner-csi-driver
   - hetzner-cloud-controller-manager
   - traefik (custom Helm values)
   - cert-manager + ClusterIssuer (Let's Encrypt)
   - galaxy (the platform Helm chart)

6. Galaxy Helm chart installs:
   - Galaxy CRDs
   - Galaxy Operator deployment
   - Galaxy API deployment
   - Galaxy Web deployment
   - PostgreSQL (for Galaxy's own DB - via CNPG)
   - All service operators (idle until user creates CRDs)
```

---

## Frontend (Web UI) Structure

Simple. Like Supabase. No K8s terminology anywhere.

```
Login / Register
│
├── Projects
│   ├── Create Project
│   │     └── Name, Region
│   │
│   └── Project Dashboard
│       │
│       ├── Overview
│       │     ├── Status (all green / issues)
│       │     ├── Quick stats (apps, DBs, storage used)
│       │     └── Monthly cost estimate
│       │
│       ├── Apps
│       │     ├── Deploy New
│       │     │     ├── From GitHub (connect repo)
│       │     │     ├── From Docker Image (paste image URL)
│       │     │     └── From Template (WordPress, Ghost, etc.)
│       │     │
│       │     └── App Detail
│       │           ├── Status + URL
│       │           ├── Logs (real-time stream)
│       │           ├── Environment Variables
│       │           ├── Domains (add custom domain)
│       │           ├── Scaling (replicas slider)
│       │           ├── Resources (CPU/RAM sliders)
│       │           └── Deployments history
│       │
│       ├── Databases
│       │     ├── Add Database
│       │     │     ├── PostgreSQL
│       │     │     ├── MySQL
│       │     │     ├── MongoDB
│       │     │     ├── Redis
│       │     │     └── Key-Value Store
│       │     │
│       │     └── Database Detail
│       │           ├── Status
│       │           ├── Connection Info (copy button)
│       │           ├── Size / Usage
│       │           ├── Backups (list + restore)
│       │           └── Logs
│       │
│       ├── Storage
│       │     ├── S3 Buckets
│       │     │     ├── Create Bucket
│       │     │     └── Bucket Detail (endpoint, keys, usage)
│       │     │
│       │     └── Volumes
│       │           ├── Create Volume
│       │           └── Volume Detail (size, attached to)
│       │
│       ├── Networking
│       │     ├── Domains (list + add)
│       │     ├── Load Balancers
│       │     ├── Firewall Rules
│       │     ├── DNS Records
│       │     ├── Floating IPs
│       │     ├── VPN Peers
│       │     └── API Gateway Routes
│       │
│       ├── Auth (Keycloak simplified)
│       │     ├── Users
│       │     ├── Roles
│       │     ├── SSO Providers (Google, GitHub, SAML)
│       │     └── API Keys
│       │
│       ├── Monitoring
│       │     ├── Dashboard (Grafana embedded)
│       │     ├── Alerts (list + create)
│       │     └── Logs (Loki query UI)
│       │
│       ├── Registry
│       │     ├── Images (list)
│       │     ├── Push instructions
│       │     └── Access tokens
│       │
│       ├── Planets (scaling)
│       │     ├── Current nodes (list with metrics)
│       │     ├── Add a Planet (size selector)
│       │     └── Remove Planet
│       │
│       └── Settings
│             ├── Project settings
│             ├── Team members (Phase 2)
│             ├── Billing
│             └── Danger zone (delete project)
│
├── Billing
│     ├── Current usage breakdown
│     ├── Invoice history
│     ├── Payment method (Stripe)
│     └── Pricing calculator
│
└── Account
      ├── Profile
      ├── API Keys
      └── Galaxy CLI setup instructions
```

---

## Go Project Structure

```
galaxy/
├── cmd/
│   ├── galaxy-operator/          # Main operator binary
│   │   └── main.go               # controller-runtime manager
│   │
│   ├── galaxy-api/               # API server binary
│   │   └── main.go               # Gin/Echo HTTP server
│   │
│   └── galaxyctl/                # CLI binary
│       └── main.go               # cobra CLI
│
├── api/                          # CRD type definitions
│   └── v1alpha1/
│       ├── project_types.go
│       ├── application_types.go
│       ├── build_types.go
│       ├── planet_types.go
│       ├── database_types.go
│       ├── objectstore_types.go
│       ├── keyvaluestore_types.go
│       ├── backuppolicy_types.go
│       ├── domain_types.go
│       ├── firewall_types.go
│       ├── network_types.go
│       ├── floatingip_types.go
│       ├── loadbalancer_types.go
│       ├── dnszone_types.go
│       ├── dnsrecord_types.go
│       ├── vpnpeer_types.go
│       ├── gateway_types.go
│       ├── cloudconnector_types.go
│       ├── registry_types.go
│       ├── authrealm_types.go
│       ├── monitoring_types.go
│       ├── logpipeline_types.go
│       ├── alertrule_types.go
│       ├── usagerecord_types.go
│       ├── groupversion_info.go
│       └── zz_generated.deepcopy.go
│
├── internal/
│   ├── controller/               # Reconcilers
│   │   ├── project_controller.go
│   │   ├── application_controller.go
│   │   ├── build_controller.go
│   │   ├── planet_controller.go
│   │   ├── database_controller.go
│   │   ├── objectstore_controller.go
│   │   ├── firewall_controller.go
│   │   ├── network_controller.go
│   │   ├── floatingip_controller.go
│   │   ├── dns_controller.go
│   │   ├── domain_controller.go
│   │   ├── loadbalancer_controller.go
│   │   ├── registry_controller.go
│   │   ├── authrealm_controller.go
│   │   ├── monitoring_controller.go
│   │   ├── logpipeline_controller.go
│   │   ├── gateway_controller.go
│   │   ├── vpnpeer_controller.go
│   │   ├── cloudconnector_controller.go
│   │   ├── backup_controller.go
│   │   └── billing_controller.go
│   │
│   ├── provider/                 # Cloud provider abstraction
│   │   ├── interface.go          # Provider interface
│   │   └── hetzner/
│   │       ├── client.go         # hcloud-go wrapper
│   │       ├── server.go         # VM operations
│   │       ├── volume.go         # Volume operations
│   │       ├── network.go        # Network operations
│   │       ├── firewall.go       # Firewall operations
│   │       ├── loadbalancer.go   # LB operations
│   │       ├── floatingip.go     # Floating IP operations
│   │       ├── dns.go            # DNS operations
│   │       └── objectstorage.go  # S3 operations
│   │
│   ├── apiserver/                # REST API
│   │   ├── server.go             # HTTP server setup
│   │   ├── middleware/
│   │   │   ├── auth.go           # JWT validation
│   │   │   └── rbac.go           # Authorization
│   │   ├── handlers/
│   │   │   ├── project.go
│   │   │   ├── application.go
│   │   │   ├── database.go
│   │   │   ├── storage.go
│   │   │   ├── networking.go
│   │   │   ├── monitoring.go
│   │   │   ├── billing.go
│   │   │   └── auth.go
│   │   └── websocket/
│   │       ├── logs.go           # Real-time log streaming
│   │       └── status.go         # Real-time status updates
│   │
│   ├── installer/                # galaxyctl install logic
│   │   ├── cluster.go            # k3s cluster creation
│   │   ├── components.go         # Helm installs
│   │   └── config.go             # Installation config
│   │
│   └── billing/                  # Usage metering
│       ├── meter.go              # Resource usage tracking
│       ├── calculator.go         # Cost calculation
│       └── stripe.go             # Stripe integration
│
├── web/                          # Frontend (Next.js)
│   ├── src/app/
│   ├── src/components/
│   ├── src/lib/
│   └── package.json
│
├── charts/                       # Helm chart
│   └── galaxy/
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── crds/                  # CRD YAML manifests
│       └── templates/
│           ├── operator.yaml
│           ├── apiserver.yaml
│           ├── web.yaml
│           ├── rbac.yaml
│           └── service-operators.yaml
│
├── config/                       # Kubebuilder config
│   ├── crd/                      # Generated CRD manifests
│   ├── rbac/                     # RBAC manifests
│   └── samples/                  # Example CRs
│
├── docs/                         # Documentation site
│   ├── getting-started/
│   ├── architecture/
│   ├── services/
│   ├── api-reference/
│   └── contributing/
│
├── test/
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── hack/                         # Dev scripts
│   ├── setup-dev.sh
│   └── run-e2e.sh
│
├── Makefile                      # Build, test, generate, deploy
├── Dockerfile                    # Multi-stage build
├── go.mod
├── go.sum
├── LICENSE                       # Apache 2.0
├── GOVERNANCE.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
└── OWNERS
```

---

## What Makes This Different From Everything Else

```
                    Manages      Full        Hetzner     Open      CNCF
                    Infra?       PaaS UI?    Native?     Source?   Path?
────────────────────────────────────────────────────────────────────────
Crossplane          Yes          No          Plugin      Yes       Yes
KubeVela            Partial      Partial     No          Yes       Yes
Backstage           No           Catalog     No          Yes       Yes
Rancher             Yes          Yes         No          Yes       No (SUSE)
OpenShift           Yes          Yes         No          No        No (RH)
Coolify             Partial      Yes         No          Yes       No
CapRover            No           Yes         No          Yes       No
Dokku               No           Partial     No          Yes       No

Galaxy/Zenith       Yes          Yes         YES         Yes       YES
────────────────────────────────────────────────────────────────────────
```

Galaxy is the only project that is:
1. A complete PaaS (not a framework/toolkit)
2. Kubernetes-native (CRDs + operators)
3. Cloud-provider-native (Hetzner first, pluggable)
4. Fully open-source (Apache 2.0)
5. On the CNCF path

---

## Management Plane - How Kubernetes Gets Upgraded Without Nightmares

> One €5 server manages everything. CAPI upgrades clusters. back-zenith is the control panel.

### The Problem

Upgrading Kubernetes is terrifying:
- K3s/K8s version upgrades can break things
- Operator upgrades (CloudNativePG, Redis, cert-manager) need coordination
- Platform updates (new Zenith version) must be applied carefully
- Who does it? The user? DevOps? A script?

### The Solution: Management Cluster + CAPI

```
┌───────────────────────────────────────────────────────────────────────┐
│                                                                       │
│  MANAGEMENT PLANE (€5/mo CX22 - 2 vCPU, 4GB RAM)                    │
│                                                                       │
│  ┌─────────────┐  ┌──────────────────┐  ┌───────────────────────┐    │
│  │   k3s       │  │  CAPI + CAPH     │  │   back-zenith         │    │
│  │ (single     │  │  (Cluster API    │  │   (admin panel)       │    │
│  │  node)      │  │   Provider       │  │                       │    │
│  │             │  │   Hetzner)       │  │   back.freezenith.com │    │
│  └─────────────┘  └────────┬─────────┘  └───────────┬───────────┘    │
│                             │                         │               │
│                             │  manages                │  controls     │
│                             ▼                         ▼               │
│  ┌──────────────────────────────────────────────────────────────┐     │
│  │                                                              │     │
│  │  WORKLOAD CLUSTER(S) - where everything actually runs        │     │
│  │                                                              │     │
│  │  ┌────────────────────────────────────────────────────────┐  │     │
│  │  │ Shared Cluster (Starter plan customers)                │  │     │
│  │  │   Zenith Operator + Service Operators + User Apps      │  │     │
│  │  │   3-20 nodes (Planets)                                 │  │     │
│  │  └────────────────────────────────────────────────────────┘  │     │
│  │                                                              │     │
│  │  ┌───────────────────┐  ┌───────────────────┐               │     │
│  │  │ Dedicated Cluster │  │ Dedicated Cluster │  ...          │     │
│  │  │ (Pro customer A)  │  │ (Pro customer B)  │               │     │
│  │  │ 2-50 nodes        │  │ 3-10 nodes        │               │     │
│  │  └───────────────────┘  └───────────────────┘               │     │
│  │                                                              │     │
│  └──────────────────────────────────────────────────────────────┘     │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
```

### CAPI (Cluster API) + CAPH (Provider Hetzner)

CAPI is the CNCF standard for managing Kubernetes cluster lifecycle declaratively.
CAPH (github.com/syself/cluster-api-provider-hetzner) is the Hetzner provider for CAPI.

**What CAPI manages:**
- Cluster creation (new workload clusters)
- Kubernetes version upgrades (rolling, zero-downtime)
- Node scaling (add/remove Planets)
- Node replacement (if a Planet dies, CAPI creates a new one)
- etcd management

**Cluster creation via CAPI:**
```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: zenith-shared
  namespace: capi-system
spec:
  clusterNetwork:
    pods:
      cidrBlocks: ["10.244.0.0/16"]
    services:
      cidrBlocks: ["10.96.0.0/12"]
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: KubeadmControlPlane
    name: zenith-shared-cp
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
    kind: HetznerCluster
    name: zenith-shared
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: HetznerCluster
metadata:
  name: zenith-shared
spec:
  controlPlaneRegions: [fsn1]
  hetznerSecret:
    name: hetzner-credentials
    key:
      hcloudToken: token
  sshKeys:
    hcloud:
      - name: zenith-key
  controlPlaneLoadBalancer:
    enabled: true
    region: fsn1
```

**Kubernetes upgrade (rolling, zero-downtime):**
```
1. back-zenith shows: "Kubernetes 1.29.4 → 1.30.2 available"
2. Platform operator clicks "Upgrade"
3. back-zenith patches CAPI MachineDeployment:
     spec.template.spec.version: "v1.30.2"
4. CAPI rolling upgrade:
     → Creates new node with K8s 1.30.2
     → Cordons old node
     → Drains pods (moved to new node)
     → Deletes old node
     → Repeats for all nodes (one at a time)
5. Zero downtime. User apps keep running.
6. back-zenith shows: ✅ "Kubernetes upgraded to 1.30.2"
```

### back-zenith - The Platform Operator Panel

**URL:** `back.freezenith.com` (or self-hosted: `back.your-domain.com`)
**Auth:** Separate credentials (not connected to user-facing auth)
**Who uses it:** The person/team running the Zenith platform (you, the operator)
**NOT visible to:** End users (customers deploying apps)

**What back-zenith manages:**

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  1. CLUSTERS                                                     │
│     - View all clusters (shared + dedicated)                     │
│     - Create new clusters via CAPI                               │
│     - Upgrade Kubernetes version (rolling via CAPI)              │
│     - Scale clusters (add/remove nodes)                          │
│     - View cluster health (API server, etcd, DNS, networking)    │
│                                                                  │
│  2. MODULES (Operators & Infrastructure Components)              │
│     - CloudNativePG operator (PostgreSQL)                        │
│     - Redis Operator                                             │
│     - MySQL Operator (Oracle or Percona)                         │
│     - MongoDB Operator (Percona)                                 │
│     - cert-manager                                               │
│     - Harbor (container registry)                                │
│     - Keycloak (auth)                                            │
│     - Prometheus + Grafana (monitoring)                          │
│     - Loki (logging)                                             │
│     - NATS (message queue)                                       │
│     - Linkerd (service mesh)                                     │
│     - Traefik (ingress)                                          │
│     Each module shows: current version, available version,       │
│     changelog, one-click upgrade (Helm upgrade)                  │
│                                                                  │
│  3. PLATFORM UPDATES                                             │
│     - Current Zenith version                                     │
│     - Available updates from freezenith.com                      │
│     - Changelog for each version                                 │
│     - One-click platform upgrade                                 │
│     - Rollback to previous version                               │
│                                                                  │
│  4. TENANT MANAGEMENT                                            │
│     - List all tenants (projects)                                │
│     - Resource usage per tenant                                  │
│     - Quotas and limits                                          │
│     - Suspend/activate tenants                                   │
│                                                                  │
│  5. INFRASTRUCTURE OVERVIEW                                      │
│     - Total Hetzner resources (VMs, volumes, LBs, IPs)           │
│     - Total cost (Hetzner billing)                               │
│     - Capacity planning (% used vs available)                    │
│                                                                  │
│  6. AUDIT LOG                                                    │
│     - Who upgraded what, when                                    │
│     - Cluster events                                             │
│     - Failed operations                                          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### How Updates Flow

```
freezenith.com                    back-zenith                     Workload Cluster
(release server)                  (management plane)               (user workloads)
      │                                 │                                │
      │  1. Publishes releases          │                                │
      │  (JSON manifest + Helm charts)  │                                │
      │                                 │                                │
      ├────────────────────────────────▶│  2. Polls for updates          │
      │   GET /api/releases/latest      │     every 6 hours              │
      │                                 │                                │
      │   Response:                     │  3. Shows notification:        │
      │   {                             │     "Zenith v1.3 available"    │
      │     "version": "1.3.0",         │     "CloudNativePG 1.23 avail"│
      │     "changelog": "...",         │     "K8s 1.30.2 available"    │
      │     "chart": "oci://...",       │                                │
      │     "modules": {                │                                │
      │       "cnpg": "1.23.0",         │  4. Operator clicks "Upgrade"  │
      │       "redis": "7.2.0",         │                                │
      │       ...                       │                                │
      │     },                          │                                │
      │     "k8s_versions": [           │                                │
      │       "1.29.4", "1.30.2"        │                                │
      │     ]                           │                                │
      │   }                             │                                │
      │                                 │                                │
      │                                 ├───────────────────────────────▶│
      │                                 │  5. Applies updates:           │
      │                                 │     helm upgrade zenith ...    │
      │                                 │     helm upgrade cnpg ...      │
      │                                 │     CAPI: patch version 1.30.2 │
      │                                 │                                │
      │                                 │  6. Verifies health            │
      │                                 │◀───────────────────────────────│
      │                                 │     All pods healthy?          │
      │                                 │     API responding?            │
      │                                 │     ✅ Update complete          │
      │                                 │                                │
```

### Module Update Flow (Example: CloudNativePG 1.22 → 1.23)

```
1. back-zenith shows:
   ┌─────────────────────────────────────────────────┐
   │ CloudNativePG                                   │
   │ Current: v1.22.1    Available: v1.23.0          │
   │                                                 │
   │ Changelog:                                      │
   │ • Online PG major version upgrades              │
   │ • Improved backup performance                   │
   │ • Fixed WAL archiving edge case                 │
   │                                                 │
   │ ⚠ This updates the PostgreSQL operator.         │
   │   Running databases will be reconciled with     │
   │   the new operator version. No downtime for     │
   │   existing databases.                           │
   │                                                 │
   │           [Skip]  [Update to v1.23.0]           │
   └─────────────────────────────────────────────────┘

2. Operator clicks "Update"

3. back-zenith runs on workload cluster:
   helm upgrade cloudnative-pg \
     cnpg/cloudnative-pg \
     --version 0.23.0 \
     --namespace cnpg-system \
     --wait

4. New operator pod rolls out (old pod terminated)

5. Operator reconciles all existing PostgreSQL clusters
   (no downtime - just applies new operator logic)

6. back-zenith shows: ✅ CloudNativePG v1.23.0
```

### Platform Upgrade Flow (Zenith v1.2 → v1.3)

```
1. back-zenith polls freezenith.com:
   GET https://freezenith.com/api/releases/latest
   → { "version": "1.3.0", "chart_url": "oci://registry.freezenith.com/charts/zenith:1.3.0" }

2. back-zenith shows:
   ┌─────────────────────────────────────────────────┐
   │ 🆕 Zenith v1.3.0 available                      │
   │ Current: v1.2.1                                 │
   │                                                 │
   │ What's new:                                     │
   │ • MongoDB support (new DatabaseController)      │
   │ • Cloud Connections (AWS/GCP VPN tunnels)       │
   │ • GitOps mode (zen export/apply)                │
   │ • 47 bug fixes                                  │
   │                                                 │
   │ Breaking changes: None                          │
   │ CRD changes: 3 new CRDs added (auto-applied)   │
   │                                                 │
   │ Tested with: K8s 1.28-1.30                      │
   │                                                 │
   │           [Release Notes]  [Upgrade to v1.3.0]  │
   └─────────────────────────────────────────────────┘

3. Operator clicks "Upgrade"

4. back-zenith runs on workload cluster:
   # Update CRDs first (new CRDs added safely)
   kubectl apply -f <new CRD manifests>

   # Upgrade Zenith components (operator, API, web)
   helm upgrade zenith \
     oci://registry.freezenith.com/charts/zenith \
     --version 1.3.0 \
     --namespace zenith-system \
     --wait

5. Rolling update: new operator, API, and web pods replace old ones
   (zero downtime - K8s rolling strategy)

6. back-zenith verifies:
   - Operator pod healthy
   - API responding at /healthz
   - Web UI loading
   - All CRDs registered
   - ✅ Upgrade complete

7. If anything fails:
   helm rollback zenith 0 --namespace zenith-system
   → Instant rollback to previous version
```

### State Architecture - Always Know What You Have

> State is sacred. The platform operator must ALWAYS know exactly what's installed, what version, what's running.

Three layers of state, each with a clear purpose:

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  LAYER 1: CAPI CRDs (management k3s etcd)                         │
│  ─────────────────────────────────────────                         │
│  Source of truth for: clusters, nodes, K8s versions                │
│  Stored in: management k3s etcd (built-in)                         │
│  Examples:                                                          │
│    Cluster "zenith-shared" → K8s v1.30.2, 8 nodes, fsn1           │
│    Cluster "pro-acme"      → K8s v1.29.4, 4 nodes, nbg1           │
│  Backed up: etcd snapshot every 6 hours                            │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  LAYER 2: Zenith CRDs (workload cluster etcd)                     │
│  ─────────────────────────────────────────────                     │
│  Source of truth for: user resources (apps, DBs, domains, etc.)    │
│  Stored in: workload cluster etcd (per cluster)                    │
│  Examples:                                                          │
│    Application "user-service" → 3 replicas, github.com/repo       │
│    Database "users-db"        → PostgreSQL 16, 20GB, daily backups │
│  Backed up: etcd snapshot every 6 hours                            │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  LAYER 3: SQLite (management server)                               │
│  ────────────────────────────────────                              │
│  Source of truth for: operational metadata                          │
│  Stored in: /var/lib/back-zenith/state.db (single file)            │
│  Contains:                                                          │
│    - Audit log (who did what, when)                                │
│    - Update history (which versions installed, when upgraded)       │
│    - Module inventory (what operators installed, what version)      │
│    - Tenant metadata (quotas, billing status, notes)               │
│    - Platform config (Hetzner token ref, domain, admin settings)   │
│  Why SQLite: zero dependencies, single file backup, embedded       │
│  Backed up: copy to Hetzner Object Storage every 6 hours          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Why three layers, not one big database?**
- Layer 1 (CAPI) = Kubernetes-native, CAPI manages it. We don't reinvent this.
- Layer 2 (Zenith CRDs) = Kubernetes-native, operators reconcile it. Standard.
- Layer 3 (SQLite) = Operational data that doesn't belong in CRDs (audit, history). Zero dependencies.

**State backup:**
```
Every 6 hours (automatic cron on management server):

  1. etcd snapshot (management k3s):
     k3s etcd-snapshot save --name mgmt-$(date +%Y%m%d-%H%M)
     → uploaded to s3://zenith-backups/mgmt-etcd/

  2. SQLite backup:
     sqlite3 /var/lib/back-zenith/state.db ".backup /tmp/state.db"
     → uploaded to s3://zenith-backups/back-zenith-state/

  3. Workload cluster etcd snapshots (triggered via CAPI):
     → uploaded to s3://zenith-backups/workload-etcd/{cluster-name}/

  Retention: 30 days

  If management server dies:
    1. Create new CX22 (1 minute)
    2. Install k3s (1 minute)
    3. Restore etcd snapshot → CAPI knows all clusters again (1 minute)
    4. Restore SQLite → audit log, history recovered (10 seconds)
    5. CAPI reconciles → reconnects to workload clusters (1 minute)
    Total recovery: ~5 minutes
    Workload clusters were NEVER affected (they run independently)
```

### `zen install` - Two-Phase Installation

Phase A is automated (CLI). Phase B is interactive (browser).
This is better than all-at-once because the admin can SEE what's happening.

**Phase A: CLI creates management plane (~3 minutes)**

```bash
# User runs this on their laptop (or any machine with internet)
curl -sfL https://get.freezenith.com | sh -s -- \
  --hetzner-token=xxxxxxxxxxxxx \
  --domain=myplatform.com \
  --admin-email=admin@company.com

# What happens:
# 1. Validates Hetzner token (API call)
# 2. Creates CX22 server in Hetzner (cheapest, €4.49/mo)
# 3. SSHs into server, installs k3s
# 4. Installs CAPI + CAPH (Cluster API Provider Hetzner)
# 5. Installs back-zenith (API + web panel)
# 6. Creates DNS record: back.myplatform.com → management server IP
# 7. Sets up SSL (cert-manager + Let's Encrypt)
# 8. Creates admin account (password shown in terminal OR emailed)
#
# Output:
# ✅ Management plane ready!
#
# 🔗 Open: https://back.myplatform.com
# 👤 Username: admin
# 🔑 Password: xK9m2pL7qR4
#
# Next: Open the URL above to set up your platform.
```

**Phase B: Welcome wizard in back-zenith (~5 minutes)**

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Welcome to Zenith                                            │
│                                                                  │
│     Let's set up your platform. This takes about 5 minutes.      │
│                                                                  │
│     Step 1 of 3: Choose a region                                 │
│                                                                  │
│     Where should your platform run?                              │
│                                                                  │
│     ● Falkenstein, Germany (fsn1)     ← lowest latency to EU     │
│     ○ Nuremberg, Germany (nbg1)                                  │
│     ○ Helsinki, Finland (hel1)                                   │
│     ○ Ashburn, USA (ash)                                         │
│     ○ Hillsboro, USA (hil)                                       │
│                                                                  │
│                                              [Next →]             │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Step 2 of 3: Choose your platform size                       │
│                                                                  │
│     How many resources do you need to start?                     │
│     You can always add more later.                               │
│                                                                  │
│     ○ Starter (3 nodes)                                          │
│       6 vCPU, 12GB RAM, ~30 apps                                │
│       €13.47/mo                                                  │
│                                                                  │
│     ● Standard (5 nodes)                   ← recommended         │
│       10 vCPU, 20GB RAM, ~80 apps                               │
│       €22.45/mo                                                  │
│                                                                  │
│     ○ Large (8 nodes)                                            │
│       16 vCPU, 32GB RAM, ~150 apps                              │
│       €35.92/mo                                                  │
│                                                                  │
│     ○ Custom (choose exact sizes)                                │
│                                                                  │
│                                     [← Back]  [Next →]           │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Step 3 of 3: Confirm & Create                                │
│                                                                  │
│     Region:    Falkenstein (fsn1)                                │
│     Size:      Standard (5x CX22)                                │
│     Cost:      ~€27/mo (5 nodes + management)                   │
│     Domain:    app.myplatform.com                                │
│                                                                  │
│     What happens next:                                           │
│     1. CAPI creates 5 servers on Hetzner (~2 min)                │
│     2. Kubernetes cluster formed (~1 min)                        │
│     3. Zenith platform installed (~2 min)                        │
│     4. DNS configured, SSL provisioned                           │
│     5. Platform ready at app.myplatform.com                      │
│                                                                  │
│                              [← Back]  [🚀 Create Platform]     │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     Creating your platform...                                    │
│                                                                  │
│     ████████████████████░░░░░░░░░░ 55%                           │
│                                                                  │
│     ✅ Created 5 servers on Hetzner                               │
│     ✅ Kubernetes cluster formed (v1.30.2)                        │
│     🔄 Installing Zenith operators...                             │
│     ○ Configuring DNS                                            │
│     ○ Provisioning SSL certificate                               │
│                                                                  │
│     Estimated time remaining: ~2 minutes                         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│     🎉 Your platform is ready!                                    │
│                                                                  │
│     Platform URL:  https://app.myplatform.com                    │
│     Admin panel:   https://back.myplatform.com (you are here)    │
│                                                                  │
│     What's next:                                                 │
│     → Invite your first users                                    │
│     → Deploy your first app                                      │
│     → Read the getting started guide                             │
│                                                                  │
│     [Open Platform →]  [Stay in Admin Panel]                     │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### How Zenith Web Talks to back-zenith

When a user in Zenith web does something that needs infrastructure (e.g., "Add a Planet"):

```
User clicks "Add a Planet" in Zenith web
         │
         ▼
Zenith API (workload cluster)
  → Creates Planet CRD in workload cluster
         │
         ▼
Zenith Operator (workload cluster)
  → Sees Planet CRD, needs a new Hetzner server
  → Calls back-zenith API: POST /api/internal/planets
    (authenticated with cluster service account token)
         │
         ▼
back-zenith (management plane)
  → Validates request (quota check, billing check)
  → Updates CAPI MachineDeployment (add 1 replica)
  → Records in SQLite audit log
  → Returns: { "status": "provisioning", "id": "planet-xxx" }
         │
         ▼
CAPI (management plane)
  → Creates new Hetzner server
  → Installs K8s, joins workload cluster
  → Machine status: Ready
         │
         ▼
back-zenith notifies Zenith Operator
  → Planet CRD status: Ready
  → User sees: 🟢 "Planet online"
```

**State is preserved at every step:**
- CAPI CRD: knows the new node exists
- SQLite: audit log shows who requested it, when, why
- Zenith CRD: Planet status tracked end-to-end

### `zen` CLI - A Beautiful Terminal Experience

> The CLI should feel as polished as the web UI. Not just text output - a real TUI.

**Stack:**
- `cobra` - command structure + flags + completions
- `bubbletea` (charmbracelet) - interactive TUI framework
- `lipgloss` (charmbracelet) - terminal styling (colors, borders, padding)
- `bubbles` (charmbracelet) - pre-built components (spinners, tables, text input, progress bars)
- `glamour` (charmbracelet) - render markdown in terminal
- `huh` (charmbracelet) - beautiful form/wizard prompts

**Color scheme:** Emerald green (#10b981) as primary, matching the web theme.

#### `zen` (no args) - Interactive TUI Dashboard

Running just `zen` opens a full-screen interactive dashboard. Like k9s, but for Zenith.

```
┌─ zen ─────────────────────────────────────────────────────────────┐
│                                                                   │
│  ⬡ Zenith v1.2.1          myplatform.com         K8s v1.30.2    │
│  ─────────────────────────────────────────────────────────────── │
│                                                                   │
│  PROJECT: my-startup                                              │
│                                                                   │
│  Apps (12)                                                        │
│  ┌──────────────────┬────────┬──────────┬─────────┬────────────┐ │
│  │ NAME             │ STATUS │ REPLICAS │ CPU     │ MEMORY     │ │
│  ├──────────────────┼────────┼──────────┼─────────┼────────────┤ │
│  │ frontend         │ ● Run  │ 2/2      │ 120m    │ 256Mi      │ │
│  │ api-gateway      │ ● Run  │ 2/2      │ 340m    │ 412Mi      │ │
│  │ user-service     │ ● Run  │ 3/3      │ 245m    │ 380Mi      │ │
│  │ order-service    │ ● Run  │ 2/2      │ 180m    │ 290Mi      │ │
│  │▸payment-service  │ ● Run  │ 1/1      │ 90m     │ 128Mi      │ │
│  │ notification-svc │ ● Run  │ 1/1      │ 45m     │ 64Mi       │ │
│  │ build-worker     │ ○ Stop │ 0/1      │ -       │ -          │ │
│  └──────────────────┴────────┴──────────┴─────────┴────────────┘ │
│                                                                   │
│  Databases (3)           Storage (1)           Planets (5)        │
│  ● users-db  PG 16      ● uploads  4.2/10GB   ● planet-01 ● ok  │
│  ● orders-db PG 16      						   ● planet-02 ● ok  │
│  ● cache     Redis 7                           ● planet-03 ● ok  │
│                                                 ● planet-04 ● ok  │
│                                                 ● planet-05 ● ok  │
│                                                                   │
│  [Tab] switch section  [Enter] detail  [/] search  [q] quit     │
│  [l] logs  [s] scale  [d] deploy  [r] redeploy  [?] help        │
└───────────────────────────────────────────────────────────────────┘
```

Navigate with arrow keys. Press Enter on an app to see details. Press `l` to tail logs. Press `d` to deploy. Fully interactive, zero flags needed.

#### `zen install` - Animated Installation

```
$ zen install

  ⬡ Zenith Installer v1.3.0

  ┌─────────────────────────────────────────────────────────┐
  │                                                         │
  │  Hetzner Cloud Token: [••••••••••••••••••••••]  ✓      │
  │                                                         │
  │  Domain: [myplatform.com                      ]  ✓      │
  │                                                         │
  │  Admin Email: [admin@company.com              ]  ✓      │
  │                                                         │
  └─────────────────────────────────────────────────────────┘

  Press Enter to continue...

  ⠸ Creating management server on Hetzner (fsn1)...

  ✓ Management server created (116.203.xx.xx)       4s
  ✓ SSH connection established                       1s
  ✓ k3s installed                                   12s
  ✓ CAPI + CAPH installed                            8s
  ✓ back-zenith deployed                             5s
  ✓ DNS configured (back.myplatform.com)             2s
  ⠸ Waiting for SSL certificate...                  15s
  ✓ SSL provisioned (Let's Encrypt)                  3s

  ┌─────────────────────────────────────────────────────────┐
  │                                                         │
  │  ✅ Management plane ready!                              │
  │                                                         │
  │  🔗 https://back.myplatform.com                         │
  │  👤 admin                                               │
  │  🔑 xK9m2pL7qR4vBn8                                    │
  │                                                         │
  │  Open the URL above to set up your platform.            │
  │                                                         │
  └─────────────────────────────────────────────────────────┘
```

The prompts use `huh` forms (beautiful, inline, with validation). The progress uses animated spinners that turn into checkmarks. Timing shown for each step.

#### `zen deploy` - Deploy From Current Directory

```
$ cd ~/projects/my-api && zen deploy

  ⬡ Deploying to my-startup

  Detected:
    Language:   Go (go.mod found)
    Dockerfile: ✓ found
    Port:       8080 (from EXPOSE)
    Branch:     main (3 commits ahead)

  ┌─────────────────────────────────────────────────────────┐
  │  App name:  [my-api                           ]         │
  │  Replicas:  [1   ] ← → to change                       │
  │  Expose:    ○ Internal only  ● Public (with domain)     │
  │  Domain:    [api.myplatform.com               ]         │
  └─────────────────────────────────────────────────────────┘

  [Enter] Deploy   [Esc] Cancel

  ⠸ Building image...

  Step 1/8 : FROM golang:1.22-alpine AS builder
  Step 2/8 : WORKDIR /app
  Step 3/8 : COPY go.mod go.sum ./
  Step 4/8 : RUN go mod download
  ████████████████████████████████████████ 100%
  Step 5/8 : COPY . .
  Step 6/8 : RUN go build -o server .
  Step 7/8 : FROM alpine:3.19
  Step 8/8 : COPY --from=builder /app/server /server

  ✓ Image built                                    32s
  ✓ Image pushed to registry                        4s
  ✓ Deployment created                              2s
  ⠸ Waiting for pods ready... (1/1)

  ✓ Deployed!

  ┌─────────────────────────────────────────────────────────┐
  │  🚀 my-api is live!                                     │
  │                                                         │
  │  URL:      https://api.myplatform.com                   │
  │  Status:   ● Running (1 instance)                       │
  │  Logs:     zen logs my-api                              │
  │  Scale:    zen scale my-api 3                           │
  └─────────────────────────────────────────────────────────┘
```

#### `zen logs` - Color-Coded Streaming Logs

```
$ zen logs user-service --follow

  ⬡ user-service (3 instances) [Ctrl+C to stop]

  ┌─ instance-1 ──────────────────────────────────────────────────┐
  │ 14:23:01 INF  Request handled  method=GET path=/users/123     │
  │ 14:23:01 INF  Cache hit        key=user:123 ttl=245s          │
  │ 14:23:02 INF  Request handled  method=POST path=/users        │
  │ 14:23:02 WRN  Slow query       duration=342ms table=users     │
  │ 14:23:03 ERR  Connection reset  remote=10.244.0.15:8080       │
  │ 14:23:03 INF  Reconnected      remote=10.244.0.15:8080       │
  └───────────────────────────────────────────────────────────────┘

  INF = green, WRN = yellow, ERR = red, DBG = gray
  Auto-formatted JSON logs into human-readable lines
  [Tab] switch instance  [/] filter  [f] full JSON  [w] wrap
```

#### `zen status` - Rich Overview

```
$ zen status

  ⬡ my-startup                                     myplatform.com

  Apps ─────────────────────────────────────────────────────────
  ● frontend          2 instances   120m CPU   256Mi RAM   ● healthy
  ● api-gateway       2 instances   340m CPU   412Mi RAM   ● healthy
  ● user-service      3 instances   245m CPU   380Mi RAM   ● healthy
  ● order-service     2 instances   180m CPU   290Mi RAM   ● healthy
  ● payment-service   1 instance     90m CPU   128Mi RAM   ● healthy
  ○ build-worker      stopped

  Databases ────────────────────────────────────────────────────
  ● users-db       PostgreSQL 16   2.1GB/5GB    ● backed up 3h ago
  ● orders-db      PostgreSQL 16   4.8GB/10GB   ● backed up 3h ago
  ● cache          Redis 7         0.3GB/2GB    ● no backup (cache)

  Planets ──────────────────────────────────────────────────────
  ● planet-01 (CX22)  CPU ████████░░ 78%  RAM █████░░░░░ 52%
  ● planet-02 (CX22)  CPU ██████░░░░ 61%  RAM ██████░░░░ 63%
  ● planet-03 (CX22)  CPU █████░░░░░ 45%  RAM ████░░░░░░ 41%
  ● planet-04 (CX22)  CPU ██████░░░░ 58%  RAM █████░░░░░ 55%
  ● planet-05 (CX22)  CPU ███░░░░░░░ 32%  RAM ███░░░░░░░ 28%

  Cost: ~€27.40/mo (5 planets + management)
```

#### `zen db connect` - Instant Database Shell

```
$ zen db connect users-db

  ⬡ Connecting to users-db (PostgreSQL 16)...
  ✓ Tunnel established (local:15432 → users-db:5432)

  psql (16.2)
  Type "help" for help.

  users=> SELECT count(*) FROM users;
   count
  -------
   12847
  (1 row)

  users=> \q

  ✓ Tunnel closed
```

Automatically creates a port-forward, opens psql/redis-cli/mongosh. Zero config.

#### `zen top` - Real-Time Resource Monitor (like htop)

```
$ zen top

  ⬡ my-startup                               Refresh: 2s  [q] quit

  CPU ████████████░░░░░░░░ 58%    RAM █████████░░░░░░░░░░░ 49%

  APP                 INSTANCES  CPU        MEMORY     NET IN    NET OUT
  ──────────────────────────────────────────────────────────────────────
  api-gateway         2/2        340m/1000m 412Mi/1Gi  12.4KB/s  8.2KB/s
  user-service        3/3        245m/1500m 380Mi/1.5G  8.1KB/s  5.3KB/s
  order-service       2/2        180m/1000m 290Mi/1Gi   6.2KB/s  4.1KB/s
  frontend            2/2        120m/1000m 256Mi/512M  15.8KB/s 42.1KB/s
  payment-service     1/1         90m/500m  128Mi/512M   2.1KB/s  1.8KB/s
  notification-svc    1/1         45m/500m   64Mi/256M   0.8KB/s  0.3KB/s

  DB                  CONNECTIONS  STORAGE    QPS     SLOW QUERIES
  ──────────────────────────────────────────────────────────────────────
  users-db (PG)       23/100       2.1/5GB   145     2 (>100ms)
  orders-db (PG)      18/100       4.8/10GB  287     0
  cache (Redis)       45/1000      0.3/2GB   1.2k   -

  [↑↓] navigate  [Enter] detail  [s] sort  [f] filter
```

Live-updating every 2 seconds. Bars animate. Red highlights when resources are high.

#### `zen diff` & `zen export` - GitOps With Syntax Highlighting

```
$ zen export project my-startup --dir ./infra

  ⬡ Exporting my-startup

  ✓ infra/project.yaml
  ✓ infra/apps/frontend.yaml
  ✓ infra/apps/api-gateway.yaml
  ✓ infra/apps/user-service.yaml
  ✓ infra/apps/order-service.yaml
  ✓ infra/apps/payment-service.yaml
  ✓ infra/apps/notification-svc.yaml
  ✓ infra/databases/users-db.yaml
  ✓ infra/databases/orders-db.yaml
  ✓ infra/databases/cache.yaml
  ✓ infra/storage/uploads.yaml
  ✓ infra/networking/domains.yaml

  Exported 12 resources to ./infra/

$ zen diff -f ./infra/

  ⬡ Comparing ./infra/ with live cluster

  apps/user-service.yaml
  ─────────────────────
   spec:
  -  replicas: 3
  +  replicas: 5
     resources:
  -    cpu: "500m"
  +    cpu: "750m"
  -    memory: "512Mi"
  +    memory: "1Gi"

  databases/orders-db.yaml
  ────────────────────────
   spec:
  -  storage: 10Gi
  +  storage: 20Gi

  Summary: 2 resources changed, 10 unchanged
  Run 'zen apply -f ./infra/' to apply changes
```

Diff output uses red/green coloring like git diff but with YAML syntax highlighting.

#### `zen wizard` - Interactive Resource Creator

For users who don't remember flags:

```
$ zen wizard

  ⬡ What would you like to create?

  > Deploy an app
    Create a database
    Add a domain
    Add a planet
    Set up monitoring
    Configure a firewall

  ────────────────────────────────
  [↑↓] navigate  [Enter] select

  (selecting "Deploy an app" starts the interactive deploy wizard
   with the same steps as `zen deploy` but without needing to be
   in a project directory - prompts for everything)
```

#### `zen events` - Live Event Stream

```
$ zen events --follow

  ⬡ Event stream for my-startup  [Ctrl+C to stop]

  14:23:01  ● app/user-service      Scaled 3 → 5 instances
  14:23:03  ● app/user-service      Instance user-svc-d8f4a starting
  14:23:05  ● app/user-service      Instance user-svc-d8f4a ready
  14:23:05  ● app/user-service      Instance user-svc-e9b2c starting
  14:23:08  ● app/user-service      Instance user-svc-e9b2c ready
  14:23:08  ● app/user-service      Scale complete: 5/5 ready
  14:24:00  ● db/orders-db          Backup started (daily)
  14:24:12  ● db/orders-db          Backup complete (4.8GB → s3://backups/)
  14:25:30  ● app/frontend          Deploy started (commit abc1234)
  14:25:45  ● app/frontend          Build complete (14s)
  14:25:48  ● app/frontend          Rolling update: 1/2
  14:25:52  ● app/frontend          Rolling update: 2/2
  14:25:52  ● app/frontend          Deploy complete ✓
```

Color-coded: green for success, yellow for in-progress, red for errors.

#### `zen` ASCII Art Banner

Every command starts with the subtle `⬡` (hexagon) logo. On first run or `zen version`:

```
$ zen version

     ███████╗███████╗███╗   ██╗██╗████████╗██╗  ██╗
     ╚══███╔╝██╔════╝████╗  ██║██║╚══██╔══╝██║  ██║
       ███╔╝ █████╗  ██╔██╗ ██║██║   ██║   ███████║
      ███╔╝  ██╔══╝  ██║╚██╗██║██║   ██║   ██╔══██║
     ███████╗███████╗██║ ╚████║██║   ██║   ██║  ██║
     ╚══════╝╚══════╝╚═╝  ╚═══╝╚═╝   ╚═╝   ╚═╝  ╚═╝

     The open-source PaaS for Kubernetes
     Version: 1.3.0
     https://freezenith.com
```

#### CLI Go Dependencies

```go
// go.mod
require (
    github.com/spf13/cobra          v1.8.0    // CLI structure
    github.com/charmbracelet/bubbletea v0.25.0 // TUI framework
    github.com/charmbracelet/lipgloss  v0.9.1  // Styling
    github.com/charmbracelet/bubbles   v0.18.0 // Components (spinner, table, progress, viewport)
    github.com/charmbracelet/glamour   v0.7.0  // Markdown rendering
    github.com/charmbracelet/huh       v0.3.0  // Interactive forms
    github.com/charmbracelet/log       v0.3.1  // Styled logging
    github.com/muesli/termenv          v0.15.2 // Terminal detection
)
```

#### Command Reference

```
zen                         Interactive TUI dashboard (full-screen)
zen install                 Install management plane (interactive wizard)
zen restore                 Restore from backup (disaster recovery)

zen login                   Authenticate with Zenith platform
zen auth status             Show current auth status

zen project list            List all projects
zen project create <name>   Create new project (interactive)
zen project switch <name>   Switch active project
zen project delete <name>   Delete project (confirmation required)

zen deploy                  Deploy from current directory (auto-detect)
zen deploy --image <img>    Deploy a Docker image
zen deploy --github <repo>  Deploy from GitHub repo

zen apps                    List all apps in current project
zen apps <name>             Show app detail
zen scale <app> <n>         Scale app to n instances
zen redeploy <app>          Redeploy app (latest code)
zen rollback <app>          Rollback to previous version

zen logs <app>              Stream logs (color-coded, formatted)
zen logs <app> --since 1h   Logs from last hour
zen logs <app> --json       Raw JSON logs

zen top                     Real-time resource monitor (htop-style)
zen events                  Live event stream
zen status                  Rich overview of everything

zen db list                 List databases
zen db create               Create database (interactive wizard)
zen db connect <name>       Open database shell (auto port-forward)
zen db backup <name>        Trigger manual backup
zen db restore <name>       Restore from backup (interactive)

zen domain add <domain>     Add custom domain to an app
zen domain list             List all domains

zen planet list             List all planets (nodes)
zen planet add              Add a planet (interactive size picker)
zen planet remove <name>    Remove a planet (drain + delete)

zen export                  Export project as YAML (GitOps)
zen export --dir <path>     Export to directory structure
zen apply -f <path>         Apply YAML to cluster
zen diff -f <path>          Show diff between file and live state
zen sync status             Show GitSync status

zen wizard                  Interactive menu for everything

zen config                  Show current config
zen config set <key> <val>  Set config value
zen version                 Show version info
zen update                  Self-update zen CLI
zen completion <shell>      Generate shell completions
```

### Directory Structure Update

```
cmd/
├── zenith-operator/main.go      # Workload cluster operator
├── zenith-api/main.go           # User-facing API
├── zenith-web/                   # User-facing frontend
├── back-zenith/main.go          # Platform operator API
├── back-zenith-web/             # Platform operator frontend
└── zen/main.go                  # CLI

internal/
├── ...existing...
├── backplane/                    # back-zenith backend
│   ├── server.go                # HTTP server
│   ├── handlers/
│   │   ├── clusters.go          # CAPI cluster management
│   │   ├── modules.go           # Module (operator) management
│   │   ├── updates.go           # Platform update management
│   │   ├── tenants.go           # Tenant management
│   │   ├── infra.go             # Infrastructure overview
│   │   └── audit.go             # Audit log
│   ├── capi/
│   │   ├── client.go            # CAPI client wrapper
│   │   ├── cluster.go           # Cluster lifecycle
│   │   └── upgrade.go           # K8s version upgrades
│   ├── modules/
│   │   ├── registry.go          # Module registry (what's installed)
│   │   ├── updater.go           # Helm upgrade logic
│   │   └── versions.go          # Version checking
│   └── releases/
│       ├── checker.go           # Check freezenith.com for updates
│       └── applier.go           # Apply platform updates

web-back/                         # back-zenith frontend (Next.js)
├── src/app/
│   ├── page.tsx                 # Dashboard
│   ├── clusters/
│   ├── modules/
│   ├── updates/
│   ├── tenants/
│   ├── infrastructure/
│   └── audit/
├── src/components/
└── package.json
```

---

## Hetzner Resource Mapping (Complete)

Every Hetzner resource has a Galaxy CRD:

| Hetzner Resource | Hetzner API | Galaxy CRD | User Sees |
|-----------------|-------------|-----------|-----------|
| Cloud Server | `hcloud.Server` | `Planet` | "Add a Planet" |
| Volume | `hcloud.Volume` | auto-created by `Database`/`Storage` | (hidden) |
| Load Balancer | `hcloud.LoadBalancer` | `LoadBalancer` | "Load Balancer" |
| Network | `hcloud.Network` | `Network` | "Private Network" |
| Subnet | `hcloud.NetworkSubnet` | part of `Network` | (hidden) |
| Firewall | `hcloud.Firewall` | `Firewall` | "Firewall Rules" |
| Floating IP | `hcloud.FloatingIP` | `FloatingIP` | "Static IP" |
| SSH Key | `hcloud.SSHKey` | auto-created by `Planet` | (hidden) |
| Object Storage | S3 API | `ObjectStore` | "S3 Bucket" |
| DNS Zone | Hetzner DNS API | `DNSZone` | "DNS Zone" |
| DNS Record | Hetzner DNS API | `DNSRecord` | "DNS Record" |
| StorageBox | Hetzner Robot API | `BackupPolicy` destination | "Backup Storage" |
| (external) | AWS/GCP/Azure VPN | `CloudConnector` | "Cloud Connection" |

---

## GitOps - First-Class Citizen

> Everything in Zenith is a CRD. CRDs are YAML. YAML lives in git. GitOps is native.

### Why GitOps by Default

Zenith doesn't need a "GitOps mode" - it IS GitOps by nature. Every resource the user creates through the UI becomes a CRD in Kubernetes. Those CRDs can be exported, versioned, and applied from git.

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  TWO WAYS TO USE ZENITH (both first-class):                     │
│                                                                  │
│  1. Click-Ops (UI/API)                                          │
│     User clicks "Add PostgreSQL" → API creates CRD → Done       │
│     Perfect for: getting started, small teams, prototyping       │
│                                                                  │
│  2. GitOps (YAML in git)                                        │
│     User writes/exports YAML → pushes to git → synced to K8s    │
│     Perfect for: reproducibility, audit trail, large teams       │
│                                                                  │
│  Both produce the same result: CRDs in Kubernetes.               │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### `zen export` - Export Everything as YAML

```bash
# Export entire project as YAML
zen export project my-startup > my-startup.yaml

# Export specific resources
zen export app user-service > user-service.yaml
zen export db users-db > users-db.yaml

# Export everything into a directory structure (git-friendly)
zen export project my-startup --dir ./zenith-infra/
```

Output directory structure:
```
zenith-infra/
├── project.yaml            # Project CRD
├── apps/
│   ├── frontend.yaml       # Application CRDs
│   ├── api-gateway.yaml
│   ├── user-service.yaml
│   └── order-service.yaml
├── databases/
│   ├── users-db.yaml       # Database CRDs
│   └── orders-db.yaml
├── storage/
│   └── uploads.yaml        # ObjectStore CRDs
├── networking/
│   ├── domains.yaml        # Domain CRDs
│   ├── firewall.yaml       # Firewall CRDs
│   └── gateway.yaml        # Gateway CRDs
├── auth/
│   ├── realm.yaml          # AuthRealm CRD
│   └── sso-okta.yaml       # SSO provider config
└── monitoring/
    ├── alerts.yaml          # AlertRule CRDs
    └── log-pipeline.yaml   # LogPipeline CRDs
```

### `zen apply` - Apply YAML to Cluster

```bash
# Apply a single resource
zen apply -f user-service.yaml

# Apply entire directory
zen apply -f ./zenith-infra/

# Dry-run (show what would change)
zen apply -f ./zenith-infra/ --dry-run

# Diff (show differences with current state)
zen diff -f ./zenith-infra/
```

### ArgoCD / FluxCD Integration

Zenith CRDs work with any GitOps controller out of the box:

```yaml
# ArgoCD Application pointing to Zenith CRDs in git
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-startup-infra
  namespace: argocd
spec:
  source:
    repoURL: https://github.com/startup/infra
    path: zenith-infra/
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

ArgoCD syncs the YAML → Kubernetes applies the CRDs → Zenith Operator reconciles → Hetzner resources created.

### Git Webhook Integration (Built-in)

For users who don't want ArgoCD/Flux, Zenith has a built-in git sync:

```yaml
apiVersion: zenith.dev/v1alpha1
kind: GitSync
metadata:
  name: infra-sync
  namespace: zenith-my-startup
spec:
  repo: https://github.com/startup/infra
  branch: main
  path: zenith-infra/
  interval: 60s        # poll interval (or webhook for instant)
  prune: true           # delete resources removed from git
  webhook:
    enabled: true       # GitHub webhook for instant sync
```

### Infrastructure as Code - Full Lifecycle

```
┌───────────────┐     ┌──────────────┐     ┌──────────────────┐
│               │     │              │     │                  │
│  Developer    │────▶│  Git Repo    │────▶│  Zenith Cluster  │
│  edits YAML   │     │  (source of  │     │  (desired state  │
│               │     │   truth)     │     │   applied)       │
└───────────────┘     └──────┬───────┘     └──────────────────┘
                             │
                     ┌───────┼───────┐
                     │       │       │
                     ▼       ▼       ▼
              ArgoCD    FluxCD    zen sync
              (optional) (optional) (built-in)
```

### Drift Detection

Zenith Operator continuously reconciles. If someone changes a resource via UI while git has a different version:

1. **UI-first mode** (default): UI changes are authoritative, `zen export` to update git
2. **Git-first mode** (GitSync enabled): Git is authoritative, UI changes get reverted on next sync
3. **Lock mode**: Resources managed by GitSync show a lock icon in UI, can't be edited from UI

### Environment Promotion (git-based)

```
infra-repo/
├── base/                    # shared config
│   ├── apps/
│   └── databases/
├── staging/                 # staging overrides
│   └── kustomization.yaml
└── production/              # production overrides
    └── kustomization.yaml
```

Works with Kustomize overlays. Same CRDs, different values per environment.
