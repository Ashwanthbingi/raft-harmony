# Complete Deployment Guide: Raft KV Store

**Date:** 2026-07-11  
**Target:** Complete step-by-step deployment from scratch  

---

## Table of Contents

1. [Prerequisites & Environment Setup](#prerequisites)
2. [Getting the Code](#getting-code)
3. [Building the Backend](#building-backend)
4. [Running Locally (Single Node)](#running-locally)
5. [Running 3-Node Cluster](#running-cluster)
6. [Docker Deployment](#docker-deployment)
7. [Monitoring Stack](#monitoring)
8. [Testing & Verification](#testing)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites & Environment Setup {#prerequisites}

### Step 1.1: Install Required Software

**On Windows:**

1. **Install Go (1.21+)**
   ```powershell
   # Download from https://golang.org/dl/
   # Install to C:\Program Files\Go
   
   # Verify installation
   go version
   # Expected: go version go1.21.x windows/amd64
   ```

2. **Install Git**
   ```powershell
   # Download from https://git-scm.com/download/win
   # Install with default settings
   
   # Verify
   git --version
   ```

3. **Install Docker Desktop**
   ```powershell
   # Download from https://www.docker.com/products/docker-desktop
   # Install with default settings
   # This includes Docker Engine and Docker Compose
   
   # Verify Docker
   docker --version
   # Expected: Docker version 20.10.x or higher
   
   # Verify Docker Compose
   docker-compose --version
   # Expected: Docker Compose version 2.x or higher
   ```

4. **Install Protocol Buffers (protoc) - Optional**
   ```powershell
   # Download from https://github.com/protocolbuffers/protobuf/releases
   # Download protoc-X.X.X-win64.zip
   # Extract to C:\protoc
   # Add C:\protoc\bin to PATH environment variable
   
   # Verify
   protoc --version
   ```

### Step 1.2: Verify Your System

```powershell
# Open PowerShell and run these commands
go version
git --version
docker --version
docker-compose --version

# Check disk space (need ~5 GB for Docker images)
Get-Volume C: | Select-Object SizeRemaining
```

**Expected output:**
```
go version go1.21.0 windows/amd64
git version 2.40.0.windows.1
Docker version 20.10.17
Docker Compose version v2.10.2
```

---

## Getting the Code {#getting-code}

### Step 2.1: Clone the Repository

**Option A: Clone from GitHub (Recommended)**

```powershell
# Create a workspace directory
mkdir C:\raft-workspace
cd C:\raft-workspace

# Clone the frontend repository
git clone https://github.com/Ashwanthbingi/raft-harmony.git
cd raft-harmony

# Clone the backend repository
git clone https://github.com/Ashwanthbingi/distributed-kv-raft.git backend

# Verify structure
ls -la
# You should see:
# - raft-harmony/ (frontend)
# - backend/ (Raft KV store)
```

**Option B: If you have local copies**

```powershell
# Frontend is at: C:\Users\Aswan\copilot-worktrees\raft-harmony\ashwanthbingi-literate-parakeet
# Backend is at: E:\distributed-kv-raft-main\distributed-kv-raft-main

# You can use these directly
cd E:\distributed-kv-raft-main\distributed-kv-raft-main
```

### Step 2.2: Verify Code Structure

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Check backend structure
ls -la

# Expected folders:
# - cmd/          (main application)
# - raft/         (consensus logic)
# - kv/           (key-value store)
# - rpc/          (protobuf definitions)
# - metrics/      (monitoring)
# - scripts/      (utilities)
# - docker/       (Docker configs)

# Check files
cat go.mod        # Module definition
cat README.md     # Overview
cat Dockerfile    # Container image
```

---

## Building the Backend {#building-backend}

### Step 3.1: Install Go Dependencies

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Download all dependencies
go mod download

# Verify dependencies
go mod tidy

# List main dependencies
go mod graph
```

**Expected dependencies:**
```
github.com/prometheus/client_golang
go.etcd.io/bbolt
google.golang.org/grpc
google.golang.org/protobuf
```

### Step 3.2: Build the Binary

```powershell
# Build for Windows
go build -o raft-kv.exe ./cmd

# Verify build succeeded
ls -la raft-kv.exe
# Expected: ~10-15 MB executable

# Check that it runs
.\raft-kv.exe --help
```

**If build fails:**
```powershell
# Clean and rebuild
go clean
go build -v -o raft-kv.exe ./cmd

# Check for errors in output
```

### Step 3.3: Generate Protobuf Code (Optional)

```powershell
# Only needed if you modified rpc/raft.proto

# Generate Go code from protobuf
protoc --go_out=. --go-grpc_out=. rpc/raft.proto

# Verify generated files
ls -la rpc/*.pb.go
```

---

## Running Locally (Single Node) {#running-locally}

### Step 4.1: Start a Single Node

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Create data directory
mkdir -Force data

# Run single node (Node 1)
$env:NODE_ID = "1"
$env:HTTP_ADDR = "localhost:8000"
$env:RAFT_ADDR = "localhost:9000"
$env:PEERS = ""    # No peers for single node

.\raft-kv.exe

# Expected output:
# 2026-07-11T16:19:31Z INFO Starting Raft node 1
# 2026-07-11T16:19:31Z INFO HTTP server listening on localhost:8000
# 2026-07-11T16:19:31Z INFO Raft listening on localhost:9000
# 2026-07-11T16:19:31Z INFO Elected as leader
```

### Step 4.2: Test the Single Node

**Open a new PowerShell window:**

```powershell
# Set a key
curl -X POST http://localhost:8000/api/kv `
  -H "Content-Type: application/json" `
  -d '{"key":"hello","value":"world"}'

# Expected response:
# {"success":true,"index":1}

# Get the key
curl http://localhost:8000/api/kv/hello

# Expected response:
# {"key":"hello","value":"world","found":true}

# List all keys
curl http://localhost:8000/api/kv

# Expected response:
# {"keys":["hello"]}

# Check health
curl http://localhost:8000/health

# Expected response:
# {"status":"leader","term":1,"node_id":1}
```

### Step 4.3: Stop the Node

```powershell
# In the terminal running the node, press Ctrl+C

# Verify data was persisted
ls -la data/
# You should see: raft_1.db (BoltDB database)
```

---

## Running 3-Node Cluster {#running-cluster}

### Step 5.1: Prepare Environment

```powershell
# Create workspace for 3-node cluster
mkdir -Force C:\raft-cluster
cd C:\raft-cluster

# Copy binary and resources
Copy-Item E:\distributed-kv-raft-main\distributed-kv-raft-main\raft-kv.exe .
Copy-Item E:\distributed-kv-raft-main\distributed-kv-raft-main\MONITORING.md .

# Create data directories
mkdir -Force node1\data
mkdir -Force node2\data
mkdir -Force node3\data
```

### Step 5.2: Start Node 1 (Leader)

**Terminal 1:**
```powershell
cd C:\raft-cluster\node1

$env:NODE_ID = "1"
$env:HTTP_ADDR = "localhost:8001"
$env:RAFT_ADDR = "localhost:9001"
$env:PEERS = "localhost:9002,localhost:9003"

..\raft-kv.exe

# Expected: Node 1 started on ports 8001 (HTTP) and 9001 (Raft)
```

### Step 5.3: Start Node 2 (Follower)

**Terminal 2:**
```powershell
cd C:\raft-cluster\node2

$env:NODE_ID = "2"
$env:HTTP_ADDR = "localhost:8002"
$env:RAFT_ADDR = "localhost:9002"
$env:PEERS = "localhost:9001,localhost:9003"

..\raft-kv.exe

# Expected: Node 2 started on ports 8002 and 9002
# Will sync from Node 1
```

### Step 5.4: Start Node 3 (Follower)

**Terminal 3:**
```powershell
cd C:\raft-cluster\node3

$env:NODE_ID = "3"
$env:HTTP_ADDR = "localhost:8003"
$env:RAFT_ADDR = "localhost:9003"
$env:PEERS = "localhost:9001,localhost:9002"

..\raft-kv.exe

# Expected: Node 3 started on ports 8003 and 9003
# Will sync from Node 1
```

### Step 5.5: Verify Cluster is Running

```powershell
# Check leader election
curl http://localhost:8001/health
# Should show: {"status":"leader",...}

curl http://localhost:8002/health
# Should show: {"status":"follower",...}

curl http://localhost:8003/health
# Should show: {"status":"follower",...}
```

### Step 5.6: Test Write & Read

```powershell
# Write to leader (Node 1)
curl -X POST http://localhost:8001/api/kv `
  -H "Content-Type: application/json" `
  -d '{"key":"test1","value":"data1"}'

# Read from follower (Node 2)
curl http://localhost:8002/api/kv/test1
# Should return: {"key":"test1","value":"data1","found":true}

# Read from other follower (Node 3)
curl http://localhost:8003/api/kv/test1
# Should also work! Data is replicated
```

### Step 5.7: Test Fault Tolerance

```powershell
# Stop Node 1 (Ctrl+C in Terminal 1)
# Leader election should happen automatically

# Wait 1-2 seconds...

# Check Node 2
curl http://localhost:8002/health
# Should now show leader

# Writes still work!
curl -X POST http://localhost:8002/api/kv `
  -H "Content-Type: application/json" `
  -d '{"key":"test2","value":"data2"}'

# Start Node 1 again and it will sync
```

---

## Docker Deployment {#docker-deployment}

### Step 6.1: Setup Docker Environment

```powershell
# Verify Docker is running
docker ps
# Should return empty list (no containers running yet)

# Verify Docker Compose
docker-compose --version
```

### Step 6.2: Build Docker Image

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Build the production image
docker build -t raft-kv:latest .

# Verify image was created
docker images | grep raft-kv
# Expected: raft-kv    latest    <hash>    <size>  50MB
```

### Step 6.3: Deploy 3-Node Cluster with Docker

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Start the full cluster (3 nodes)
docker-compose up -d

# Expected output:
# Creating raft-harmony_node1_1 ... done
# Creating raft-harmony_node2_1 ... done
# Creating raft-harmony_node3_1 ... done

# Verify containers are running
docker-compose ps

# Expected output:
# NAME        STATUS
# node1       Up (healthy)
# node2       Up (healthy)
# node3       Up (healthy)
```

### Step 6.4: Test Docker Cluster

```powershell
# Write to Node 1
curl -X POST http://localhost:8001/api/kv `
  -H "Content-Type: application/json" `
  -d '{"key":"docker-test","value":"success"}'

# Read from Node 2
curl http://localhost:8002/api/kv/docker-test

# Expected: {"key":"docker-test","value":"success","found":true}

# Check logs
docker-compose logs node1
docker-compose logs node2
docker-compose logs node3
```

### Step 6.5: Deploy with Monitoring

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Stop previous deployment
docker-compose down

# Start with monitoring stack
docker-compose -f docker-compose-monitoring.yml up -d

# Wait for services to start (10-15 seconds)
Start-Sleep -Seconds 15

# Verify all services
docker-compose -f docker-compose-monitoring.yml ps

# Expected services:
# node1           - Up (healthy)
# node2           - Up (healthy)
# node3           - Up (healthy)
# prometheus      - Up
# grafana         - Up
```

### Step 6.6: Access Monitoring Stack

```powershell
# Open browser and navigate to:

# 1. Raft KV API (Node 1)
# http://localhost:8001

# 2. Prometheus (Metrics database)
# http://localhost:9090
# - Go to: Status > Targets
# - Verify all 3 nodes show "UP"

# 3. Grafana (Dashboards)
# http://localhost:3000
# - Login: admin / admin
# - Import dashboard from: grafana-replication-dashboard.json

# 4. Metrics endpoint
# http://localhost:8001/metrics
# (Will show Prometheus metrics if integrated)
```

---

## Monitoring Stack {#monitoring}

### Step 7.1: Import Grafana Dashboard

**In Grafana UI:**

```
1. Go to http://localhost:3000
2. Login with admin / admin
3. Click "+" → Dashboards → Import
4. Click "Upload JSON file"
5. Select: grafana-replication-dashboard.json
6. Select data source: Prometheus
7. Click Import
8. Dashboard appears with 4 panels:
   - Log Size
   - Replication Lag
   - Log Replication Rate
   - RPC Latency
```

### Step 7.2: Generate Metrics

```powershell
# Generate traffic to populate metrics
for ($i = 1; $i -le 100; $i++) {
    curl -X POST http://localhost:8001/api/kv `
      -H "Content-Type: application/json" `
      -d "{`"key`":`"key$i`",`"value`":`"value$i`"}"
}

# Wait 30 seconds for metrics to appear in Prometheus
Start-Sleep -Seconds 30

# Check Prometheus
# http://localhost:9090/graph
# Query: raft_log_size
# Should show increasing line
```

### Step 7.3: Check Alert Rules

```
In Prometheus UI (http://localhost:9090):

1. Go to: Alerts
2. You should see 15 alert rules:
   - NoLeaderElected
   - HighReplicationLag
   - HighRpcLatency
   - FrequentElections
   - ... (more)
3. Most should be "INACTIVE" (no alerts firing)
```

---

## Testing & Verification {#testing}

### Step 8.1: Run Unit Tests

```powershell
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# Run all tests
go test ./...

# Expected output:
# ok      distributed-kv-raft/kv      0.576s
# ok      distributed-kv-raft/raft    1.234s
# ok      distributed-kv-raft/rpc     0.345s

# Run specific test
go test -v ./kv -run TestStoreSetGet
```

### Step 8.2: Performance Test

```powershell
# Write 1000 keys
$sw = [System.Diagnostics.Stopwatch]::StartNew()

for ($i = 1; $i -le 1000; $i++) {
    curl -X POST http://localhost:8001/api/kv `
      -H "Content-Type: application/json" `
      -d "{`"key`":`"perf-test-$i`",`"value`":`"data-$i`"}" `
      -ErrorAction SilentlyContinue | Out-Null
}

$sw.Stop()
$throughput = 1000 / $sw.ElapsedMilliseconds * 1000

Write-Host "Throughput: $throughput ops/sec"
Write-Host "Latency: $($sw.ElapsedMilliseconds / 1000)s total"

# Expected: 100-500 ops/sec depending on hardware
```

### Step 8.3: Verify Data Persistence

```powershell
# Stop Docker cluster
docker-compose -f docker-compose-monitoring.yml down

# Data is persisted in volumes
docker volume ls | grep node

# Start cluster again
docker-compose -f docker-compose-monitoring.yml up -d

# Data should be restored
curl http://localhost:8001/api/kv/key1
# Should return the key you wrote earlier
```

### Step 8.4: Test Snapshot Generation

```powershell
# Write enough data to trigger snapshot
for ($i = 1; $i -le 20; $i++) {
    curl -X POST http://localhost:8001/api/kv `
      -H "Content-Type: application/json" `
      -d "{`"key`":`"snap-$i`",`"value`":`"data`"}" `
      -ErrorAction SilentlyContinue | Out-Null
}

# Check logs
docker-compose -f docker-compose-monitoring.yml logs node1

# Should see: "Snapshot generated" message
```

---

## Troubleshooting {#troubleshooting}

### Issue 1: Containers Won't Start

```powershell
# Check Docker status
docker ps -a

# View error logs
docker-compose logs

# Common causes:
# 1. Port already in use
Get-NetTCPConnection -LocalPort 8001 -ErrorAction SilentlyContinue

# Solution: Kill process using port
Get-Process | Where-Object {$_.Name -like "*raft*"} | Stop-Process -Force

# 2. Insufficient disk space
Get-Volume C: | Select-Object SizeRemaining
```

### Issue 2: Prometheus Not Scraping

```powershell
# Check Prometheus status
# http://localhost:9090/targets

# If "DOWN":
# 1. Verify containers are healthy
docker-compose -f docker-compose-monitoring.yml ps

# 2. Check container IPs
docker inspect raft-harmony_node1_1 | Select-Object -ExpandProperty NetworkSettings

# 3. Restart containers
docker-compose -f docker-compose-monitoring.yml restart
```

### Issue 3: No Leader Election

```powershell
# Check node logs
docker-compose logs node1

# Look for election messages

# If nodes can't see each other:
# 1. Check network
docker network ls | grep raft

# 2. Test connectivity between nodes
docker exec raft-harmony_node1_1 ping node2

# 3. Verify peer addresses in container
docker exec raft-harmony_node1_1 env | grep PEERS
```

### Issue 4: Data Not Replicating

```powershell
# Write to leader
curl -X POST http://localhost:8001/api/kv `
  -d '{"key":"test","value":"data"}'

# Check followers
curl http://localhost:8002/api/kv/test
# Should return the data

# If not found:
# 1. Check logs
docker-compose logs node2

# 2. Verify replication is happening
docker-compose logs node1 | grep "AppendEntries"

# 3. Check Prometheus metrics
# http://localhost:9090/graph
# Query: raft_replication_lag
```

---

## Quick Reference: Command Cheat Sheet

### Docker Commands

```powershell
# Start cluster
docker-compose -f docker-compose-monitoring.yml up -d

# Stop cluster
docker-compose -f docker-compose-monitoring.yml down

# View logs
docker-compose logs -f node1
docker-compose logs -f prometheus
docker-compose logs -f grafana

# Execute command in container
docker exec raft-harmony_node1_1 ls -la /app/data

# View resource usage
docker stats

# Remove volumes (delete all data)
docker-compose -f docker-compose-monitoring.yml down -v
```

### API Endpoints

```
POST   http://localhost:8001/api/kv              # Set key
GET    http://localhost:8001/api/kv/{key}        # Get key
GET    http://localhost:8001/api/kv              # List all keys
GET    http://localhost:8001/health              # Health check
GET    http://localhost:8001/metrics             # Prometheus metrics
```

### Web UIs

```
Raft Node 1:   http://localhost:8001
Raft Node 2:   http://localhost:8002
Raft Node 3:   http://localhost:8003
Prometheus:    http://localhost:9090
Grafana:       http://localhost:3000 (admin/admin)
```

---

## Summary: Complete Workflow

```powershell
# 1. Install prerequisites
# - Go, Git, Docker Desktop, Protobuf

# 2. Get code
cd E:\distributed-kv-raft-main\distributed-kv-raft-main

# 3. Build
go build -o raft-kv.exe ./cmd

# 4. Test locally
.\raft-kv.exe

# 5. Deploy with Docker
docker build -t raft-kv:latest .
docker-compose -f docker-compose-monitoring.yml up -d

# 6. Test APIs
curl http://localhost:8001/api/kv
curl -X POST http://localhost:8001/api/kv -d '{"key":"test","value":"data"}'

# 7. Monitor
# Open http://localhost:3000 (Grafana)
# Open http://localhost:9090 (Prometheus)
```

---

## Next Steps

✅ System running locally or in Docker  
✅ 3-node cluster operational  
✅ Monitoring stack collecting metrics  

**What to do next:**

1. **Load Testing** - Generate sustained traffic to test performance
2. **Failover Testing** - Stop nodes and verify recovery
3. **Backup Strategy** - Backup Prometheus and Grafana data volumes
4. **Alerting** - Configure Slack/email alerts
5. **Production Deployment** - Move to cloud (AWS, GCP, Azure)

---

**You now have a complete, production-grade Raft KV cluster running with full observability!**
