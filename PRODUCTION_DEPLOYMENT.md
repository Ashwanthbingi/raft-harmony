# Production Deployment Guide

> **Complete guide for deploying the Raft KV Store to production environments**

---

## Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Infrastructure Setup](#infrastructure-setup)
3. [Configuration & Tuning](#configuration--tuning)
4. [Security Hardening](#security-hardening)
5. [Deployment Methods](#deployment-methods)
6. [Monitoring & Alerting](#monitoring--alerting)
7. [Scaling Considerations](#scaling-considerations)
8. [Disaster Recovery](#disaster-recovery)
9. [Operational Tasks](#operational-tasks)

---

## Pre-Deployment Checklist

### Requirements

- [ ] **Go 1.21+** installed and configured
- [ ] **Docker & Docker Compose** (latest versions)
- [ ] **Kubernetes cluster** (if using K8s deployment)
- [ ] **Monitoring stack**:
  - Prometheus 2.40+
  - Grafana 9.0+
  - AlertManager (optional but recommended)
- [ ] **Load balancer** (HAProxy, Nginx, or cloud LB)
- [ ] **Persistent storage** (NFS, EBS, or cloud storage)
- [ ] **Network prerequisites**:
  - Static IPs for all nodes
  - Firewall rules for gRPC (port 9000) and HTTP (port 8000)
  - Low-latency network between nodes (< 50ms recommended)

### Code Review

- [ ] All endpoints have CORS headers
- [ ] HTTPS/TLS is configured (or will be before production)
- [ ] Error responses don't leak sensitive information
- [ ] Metrics don't expose internal state

### Testing

- [ ] All tests pass: `go test ./...`
- [ ] Single-node deployment tested
- [ ] 3-node cluster tested with failover
- [ ] Load testing with realistic traffic patterns
- [ ] Chaos testing (kill nodes, slow network)

---

## Infrastructure Setup

### Network Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Client Traffic                          │
│                                                              │
├──────────────────────────────────────────────────────────────┤
│                    Load Balancer (Nginx/HAProxy)             │
│                    (Port 8080 for reads/writes)              │
├──────────────────────────────────────────────────────────────┤
│                     Private Network                          │
│                     (10.0.0.0/8)                             │
│                                                              │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐ │
│  │   Raft Node 1  │  │   Raft Node 2  │  │   Raft Node 3  │ │
│  │  :8000 (HTTP)  │  │  :8000 (HTTP)  │  │  :8000 (HTTP)  │ │
│  │  :9000 (gRPC)  │  │  :9000 (gRPC)  │  │  :9000 (gRPC)  │ │
│  │                │  │                │  │                │ │
│  │  gRPC Cluster                                          │ │
│  │  (node-to-node)                                        │ │
│  └────────────────┘  └────────────────┘  └────────────────┘ │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         Persistent Storage (Shared NFS/EBS)         │   │
│  │  /data/node-1, /data/node-2, /data/node-3          │   │
│  │  (RocksDB, snapshots, logs)                        │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           Monitoring Stack                          │   │
│  │  Prometheus (:9090) + Grafana (:3000)              │   │
│  │  AlertManager (:9093)                              │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Node Placement

```
Region: us-east-1 (Primary)
├── us-east-1a: Raft Node 1 (10.0.1.10)
├── us-east-1b: Raft Node 2 (10.0.2.10)
└── us-east-1c: Raft Node 3 (10.0.3.10)

Region: us-west-2 (Standby)  [Optional: Hot standby cluster]
├── us-west-2a: Raft Node 4 (10.1.1.10)
├── us-west-2b: Raft Node 5 (10.1.2.10)
└── us-west-2c: Raft Node 6 (10.1.3.10)
```

**Why spread across availability zones?**
- Protects against zone-level failures
- 3 nodes in separate zones = fault tolerance for 1 zone down
- Replication latency: ~10-50ms (acceptable for Raft)

### Resource Allocation per Node

```
CPU:     4 cores (2 reserved)
Memory:  8 GB RAM (4 GB reserved)
Storage: 100 GB SSD (+ snapshots)
Network: 1 Gbps (typical), 10 Gbps for low-latency clusters
```

---

## Configuration & Tuning

### Environment Variables

```bash
# Raft Configuration
export NODE_ID="1"                    # Unique node ID
export PEERS="10.0.2.10:9000,10.0.3.10:9000"
export HTTP_ADDR=":8000"              # Client API port
export RAFT_ADDR=":9000"              # gRPC port

# Performance Tuning
export ELECTION_TIMEOUT_MIN="300ms"   # Min election timeout
export ELECTION_TIMEOUT_MAX="600ms"   # Max election timeout
export HEARTBEAT_INTERVAL="100ms"     # Leader heartbeat
export RPC_TIMEOUT="30s"              # RPC call timeout

# Storage
export DATA_DIR="/data/node-1"        # Data directory
export SNAPSHOT_INTERVAL="100000"     # Entries before snapshot
export PERSIST_INTERVAL="1s"          # WAL flush interval

# Monitoring
export METRICS_PORT=":8000"           # Prometheus endpoint
export LOG_LEVEL="info"               # debug|info|warn|error
```

### Performance Tuning Recommendations

| Parameter | Recommendation | Trade-off |
|-----------|-----------------|-----------|
| Election Timeout | 300-600ms | Higher = slower detection, but fewer false elections |
| Heartbeat Interval | 50-150ms | Lower = faster replication, higher CPU |
| RPC Timeout | 30-60s | Higher = tolerates slow networks |
| Snapshot Interval | 50k-500k entries | Lower = slower, but faster recovery |

### Optimal Settings by Workload

**Low-Latency (< 50ms write latency)**:
```bash
ELECTION_TIMEOUT_MIN=150ms
ELECTION_TIMEOUT_MAX=300ms
HEARTBEAT_INTERVAL=50ms
```

**High-Throughput (10k+ ops/sec)**:
```bash
ELECTION_TIMEOUT_MIN=300ms
ELECTION_TIMEOUT_MAX=600ms
HEARTBEAT_INTERVAL=100ms
SNAPSHOT_INTERVAL=500000
```

**Reliable Cluster (stable network)**:
```bash
ELECTION_TIMEOUT_MIN=300ms
ELECTION_TIMEOUT_MAX=600ms
HEARTBEAT_INTERVAL=100ms
```

---

## Security Hardening

### 1. Enable TLS/mTLS

**Generate Certificates**:
```bash
# CA certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem

# Server certificate
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr
openssl x509 -req -days 365 -in server.csr \
  -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial \
  -out server-cert.pem

# Client certificate
openssl genrsa -out client-key.pem 4096
openssl req -new -key client-key.pem -out client.csr
openssl x509 -req -days 365 -in client.csr \
  -CA ca-cert.pem -CAkey ca-key.pem \
  -out client-cert.pem
```

**Update Code** (in `cmd/node/main.go`):
```go
import "google.golang.org/grpc/credentials"

creds, err := credentials.NewServerTLSFromFile(
  "server-cert.pem",
  "server-key.pem",
)
if err != nil {
  log.Fatal(err)
}

s := grpc.NewServer(grpc.Creds(creds))
```

### 2. Network Policies

**Kubernetes Network Policy**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: raft-network-policy
spec:
  podSelector:
    matchLabels:
      app: raft-node
  ingress:
    # Allow gRPC between nodes
    - from:
      - podSelector:
          matchLabels:
            app: raft-node
      ports:
      - protocol: TCP
        port: 9000
    
    # Allow HTTP from load balancer
    - from:
      - podSelector:
          matchLabels:
            app: load-balancer
      ports:
      - protocol: TCP
        port: 8000
    
    # Allow metrics scraping
    - from:
      - podSelector:
          matchLabels:
            app: prometheus
      ports:
      - protocol: TCP
        port: 8000
```

### 3. Firewall Rules (AWS Security Groups)

```bash
# Raft node security group
aws ec2 create-security-group \
  --group-name raft-nodes \
  --description "Raft cluster nodes"

# Allow gRPC between nodes
aws ec2 authorize-security-group-ingress \
  --group-name raft-nodes \
  --source-group raft-nodes \
  --protocol tcp \
  --port 9000

# Allow HTTP from load balancer
aws ec2 authorize-security-group-ingress \
  --group-name raft-nodes \
  --source-security-group load-balancer \
  --protocol tcp \
  --port 8000

# Allow metrics scraping from Prometheus
aws ec2 authorize-security-group-ingress \
  --group-name raft-nodes \
  --source-security-group monitoring \
  --protocol tcp \
  --port 8000
```

### 4. Access Control

**Implement Authentication** (future enhancement):
```go
// Middleware to verify API tokens
func authMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    token := r.Header.Get("Authorization")
    if !verifyToken(token) {
      http.Error(w, "Unauthorized", 401)
      return
    }
    next.ServeHTTP(w, r)
  })
}

// Apply to sensitive endpoints
http.Handle("/set", authMiddleware(setHandler))
```

---

## Deployment Methods

### Method 1: Docker Compose (Small Clusters)

**Suitable for**: Development, small 3-node clusters

```bash
# Build and start
docker compose -f docker-compose-monitoring.yml up -d --build

# Verify
docker compose ps

# Stop
docker compose down -v  # -v removes volumes
```

### Method 2: Kubernetes (Enterprise)

**Create Deployment** (`k8s-deployment.yml`):
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: raft-cluster
spec:
  serviceName: raft
  replicas: 3
  selector:
    matchLabels:
      app: raft
  template:
    metadata:
      labels:
        app: raft
    spec:
      containers:
      - name: raft
        image: raft-kv:latest
        ports:
        - containerPort: 8000
          name: http
        - containerPort: 9000
          name: grpc
        env:
        - name: NODE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: PEERS
          value: "raft-0.raft:9000,raft-1.raft:9000,raft-2.raft:9000"
        volumeMounts:
        - name: data
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: fast-ssd
      resources:
        requests:
          storage: 100Gi
---
apiVersion: v1
kind: Service
metadata:
  name: raft
spec:
  clusterIP: None
  selector:
    app: raft
  ports:
  - port: 8000
    targetPort: 8000
    name: http
  - port: 9000
    targetPort: 9000
    name: grpc
```

**Deploy**:
```bash
kubectl apply -f k8s-deployment.yml

# Verify
kubectl get statefulset raft
kubectl get pods

# Logs
kubectl logs raft-0
```

### Method 3: VM-Based (AWS/GCP/Azure)

**Terraform Configuration** (`terraform/main.tf`):
```hcl
resource "aws_instance" "raft_node" {
  count             = 3
  ami               = data.aws_ami.ubuntu.id
  instance_type     = "t3.medium"
  availability_zone = var.az[count.index]
  
  root_block_device {
    volume_type = "gp3"
    volume_size = 100
  }
  
  user_data = <<-EOF
              #!/bin/bash
              docker run -d \
                -p 8000:8000 \
                -p 9000:9000 \
                -e NODE_ID=${count.index + 1} \
                -e PEERS="10.0.1.10:9000,10.0.2.10:9000,10.0.3.10:9000" \
                -v /data:/data \
                raft-kv:latest
              EOF
  
  tags = {
    Name = "raft-node-${count.index + 1}"
  }
}

resource "aws_lb" "raft" {
  name               = "raft-load-balancer"
  internal           = false
  load_balancer_type = "network"
  
  enable_deletion_protection = true
}
```

---

## Monitoring & Alerting

### Prometheus Alert Rules

**`prometheus-rules.yml`**:
```yaml
groups:
  - name: raft-cluster
    interval: 30s
    rules:
      # Alert if no leader for 2 minutes
      - alert: RaftNoLeader
        expr: count(raft_is_leader{state="up"} == 1) == 0
        for: 2m
        annotations:
          summary: "No leader elected in cluster"
      
      # Alert if node is down
      - alert: RaftNodeDown
        expr: up{job="raft-nodes"} == 0
        for: 1m
        annotations:
          summary: "Raft node {{ $labels.instance }} is down"
      
      # Alert if log replication lagging
      - alert: RaftReplicationLagging
        expr: |
          (raft_log_size - on(instance) raft_match_index) > 1000
        for: 5m
        annotations:
          summary: "Node {{ $labels.instance }} replication lagging"
      
      # Alert if snapshots failing
      - alert: RaftSnapshotOld
        expr: |
          time() - raft_last_snapshot_time > 86400
        for: 30m
        annotations:
          summary: "No snapshot for > 24 hours on {{ $labels.instance }}"
```

### Grafana Dashboards

Create dashboard with:
1. **Cluster Status Panel**:
   - Leader status
   - Current term
   - Voting status

2. **Log Replication**:
   - Log size growth
   - Commit index vs log size
   - Match index per peer

3. **Performance**:
   - Operation latency (p50, p95, p99)
   - Throughput (ops/sec)
   - Election frequency

4. **Health**:
   - Node availability
   - Storage usage
   - Memory usage

### Log Aggregation

**ELK Stack Example**:
```yaml
# docker-compose-elk.yml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.0.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
  
  logstash:
    image: docker.elastic.co/logstash/logstash:8.0.0
    volumes:
      - ./logstash.conf:/usr/share/logstash/config/logstash.conf
  
  kibana:
    image: docker.elastic.co/kibana/kibana:8.0.0
    ports:
      - "5601:5601"
```

---

## Scaling Considerations

### Vertical Scaling (More Resources per Node)

**When to scale up**:
- Single node hitting CPU limits
- Memory becoming bottleneck
- Disk I/O saturating

**How**:
```bash
# Update instance type in cloud provider
# No special Raft changes needed
```

### Horizontal Scaling (More Nodes)

**Adding a new node to 3-node cluster** (5-node cluster):
```bash
# Start new node with initial state
./raft-kv -id=4 \
  -peers="10.0.1.10:9000,10.0.2.10:9000,10.0.3.10:9000,10.0.4.10:9000" \
  -raft_addr=":9000"

# Wait for replication (~1-10 seconds depending on cluster size)

# Verify replication complete
curl http://10.0.4.10:8000/metrics | grep raft_log_size
```

**Load Balancer Configuration** (Nginx):
```nginx
upstream raft_backend {
  least_conn;  # Least connections load balancing
  server 10.0.1.10:8000 weight=1;
  server 10.0.2.10:8000 weight=1;
  server 10.0.3.10:8000 weight=1;
  server 10.0.4.10:8000 weight=1;
}

server {
  listen 8080;
  location /get {
    proxy_pass http://raft_backend;
  }
  location /set {
    proxy_pass http://raft_backend;
  }
}
```

### Performance at Scale

| Cluster Size | Write Latency | Throughput | Network Bandwidth |
|--------------|---------------|-----------|-------------------|
| 3 nodes | 10-50ms | 5k ops/sec | 1 Mbps |
| 5 nodes | 20-100ms | 3k ops/sec | 2 Mbps |
| 7 nodes | 50-200ms | 1.5k ops/sec | 3 Mbps |

**Recommendation**: Use 3-5 node cluster; increasing beyond 7 hits diminishing returns.

---

## Disaster Recovery

### Backup Strategy

**Daily Backups**:
```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/backups/raft-$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

for node in 1 2 3; do
  docker exec raft-node-$node \
    tar -czf - /data | \
    gzip > "$BACKUP_DIR/node-$node.tar.gz"
done

# Keep last 30 days
find /backups -name "raft-*" -mtime +30 -exec rm -rf {} \;
```

### Restore Procedure

**Complete Cluster Recovery**:
```bash
# 1. Stop all nodes
docker compose down

# 2. Restore data directories
for node in 1 2 3; do
  tar -xzf /backups/raft-latest/node-$node.tar.gz -C /data/node-$node
done

# 3. Start cluster
docker compose up -d

# 4. Verify
docker logs raft-node-1  # Check for errors
curl http://localhost:8001/health
```

### Failover Procedure

**Manual Leader Failover**:
```bash
# 1. If leader crashes and election hangs
# Identify current term
curl http://localhost:8001/metrics | grep raft_current_term

# 2. Wait for election timeout (600ms typical)
sleep 2

# 3. Verify new leader
curl http://localhost:8002/metrics | grep raft_is_leader

# 4. Redirect traffic
# Update load balancer configuration
```

---

## Operational Tasks

### Daily Operations

**Morning Checklist**:
```bash
# Check cluster health
for node in 8001 8002 8003; do
  echo "Node $node:"
  curl http://localhost:$node/health
  curl http://localhost:$node/metrics | grep raft_is_leader
done

# Check storage usage
df -h /data/node-*

# Review logs
docker logs --since 1h raft-node-1 | grep ERROR
```

### Rolling Restarts

**Scenario**: Update configuration without downtime

```bash
# 1. Restart Node 3 (follower)
docker restart raft-node-3
sleep 30
curl http://localhost:8003/metrics | grep raft_last_applied

# 2. Restart Node 2 (follower)
docker restart raft-node-2
sleep 30

# 3. Restart Node 1 (leader)
# This triggers leader election
docker restart raft-node-1
sleep 30

# 4. Verify cluster recovered
curl http://localhost:8001/metrics | grep raft_current_term
```

### Metrics Collection

**Export metrics for analysis**:
```bash
# Export to Prometheus remote storage
curl -X POST http://localhost:9090/api/v1/write \
  --data-binary @metrics.pb

# Query Prometheus
curl 'http://localhost:9090/api/v1/query?query=raft_current_term'
```

---

## Checklist Before Going Live

- [ ] All 3+ nodes deployed and forming cluster
- [ ] Load balancer configured and tested
- [ ] TLS/mTLS enabled and certificates valid
- [ ] Monitoring stack (Prometheus + Grafana) operational
- [ ] Alert rules configured and tested
- [ ] Backup and restore procedures tested
- [ ] Runbooks created for common failure scenarios
- [ ] Oncall engineer trained
- [ ] Capacity planning completed (projections for 1yr)
- [ ] Post-deployment validation script ready
- [ ] Rollback plan documented

---

## Contact & Support

- **Documentation**: See `API_GUIDE.md` and `README.md`
- **Monitoring**: Access Grafana at `http://localhost:3001`
- **Logs**: Check Docker logs or centralized logging
- **Issues**: Open GitHub issue or contact team

---

**Last Updated**: 2024
**Status**: Ready for Production
