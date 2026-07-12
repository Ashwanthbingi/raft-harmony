# Docker Quick Start Guide

**File:** `DOCKER_QUICKSTART.md`

## 30-Second Setup

### Prerequisites
- Docker installed (https://www.docker.com/get-started)
- Docker Compose (comes with Docker Desktop)
- ~2GB free disk space

### One Command to Start

```bash
cd E:\distributed-kv-raft-main\distributed-kv-raft-main
docker-compose up -d
```

**Expected Output:**
```
Creating network "distributed-kv-raft_raft-network" with driver "bridge"
Creating raft-node-1 ... done
Creating raft-node-2 ... done
Creating raft-node-3 ... done
```

### Verify Cluster is Running

```bash
docker-compose ps
```

**Expected Output:**
```
NAME               STATUS              PORTS
raft-node-1        Up 2 seconds        0.0.0.0:8001->8000/tcp, 0.0.0.0:9001->9000/tcp
raft-node-2        Up 2 seconds        0.0.0.0:8002->8000/tcp, 0.0.0.0:9002->9000/tcp
raft-node-3        Up 2 seconds        0.0.0.0:8003->8000/tcp, 0.0.0.0:9003->9000/tcp
```

---

## Test Cluster

### Health Check

```bash
curl http://localhost:8001/health
# Expected: HTTP 200 OK
```

### Write Data (Node 1)

```bash
curl -X POST http://localhost:8001/api/set \
  -H "Content-Type: application/json" \
  -d '{"key": "test", "value": "hello world"}'
```

### Read from Different Node (Node 2)

```bash
curl http://localhost:8002/api/get?key=test
# Expected: {"value": "hello world"}
```

### Read from Third Node (Node 3)

```bash
curl http://localhost:8003/api/get?key=test
# Expected: {"value": "hello world"}
```

---

## Common Operations

### View Logs

```bash
# All nodes
docker-compose logs -f

# Specific node
docker-compose logs -f node1
```

### Stop Cluster

```bash
docker-compose stop
```

### Restart Cluster

```bash
docker-compose restart
```

### Completely Remove Cluster (including volumes)

```bash
docker-compose down -v
```

### Rebuild after Code Changes

```bash
docker-compose up -d --build
```

---

## Test Failover

### Kill Node 1 (Simulate Leader Failure)

```bash
docker stop raft-node-1
```

### Cluster auto-elects new leader

Wait 5 seconds for election. Other nodes continue working:

```bash
curl http://localhost:8002/api/get?key=test
# Still works! Node 2 or 3 is now leader
```

### Restart Node 1

```bash
docker start raft-node-1
```

Node 1 rejoins cluster and catches up automatically.

---

## Troubleshooting

### "Cannot connect to Docker daemon"

**Solution:** Start Docker Desktop

### "Port already in use: 8001"

**Solution:** Change port mapping in docker-compose.yml
```yaml
node1:
  ports:
    - "8011:8000"  # Changed from 8001 to 8011
```

### Container exits immediately

```bash
docker-compose logs node1
# Read error message and fix
```

### Cluster won't form

```bash
# Check if nodes can communicate
docker exec raft-node-1 ping node2
# Should return ping stats, not "unknown host"
```

---

## Performance Notes

- **First build:** 30-60s (downloads Go SDK)
- **Subsequent builds:** 5-10s (cached layers)
- **Cluster startup:** ~2-3 seconds
- **Node recovery:** <1 second
- **API response:** <10ms

---

## Next Steps

1. Read full documentation: `DOCKER.md`
2. Run manual tests: See `TESTING.md`
3. Deploy to production: Follow `DOCKER.md` deployment guide
4. Add monitoring: See Task 6 (Monitoring & Metrics)

---

**That's it! Cluster is running.** 🎉

