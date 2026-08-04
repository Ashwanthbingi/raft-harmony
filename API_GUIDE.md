# API Documentation - Distributed Raft Key-Value Store

> **Comprehensive guide for interacting with the Raft consensus-based KV store**

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [API Reference](#api-reference)
4. [Cluster Architecture](#cluster-architecture)
5. [Consistency Guarantees](#consistency-guarantees)
6. [Error Handling](#error-handling)
7. [Examples](#examples)
8. [Monitoring](#monitoring)
9. [Troubleshooting](#troubleshooting)

---

## Overview

The Distributed Raft KV Store provides a simple yet powerful REST API for storing and retrieving key-value pairs in a fault-tolerant cluster. The backend ensures **strong consistency** through the Raft consensus algorithm.

### Key Features

- **Strong Consistency**: All writes are linearizable
- **Fault Tolerance**: Automatic failover when nodes crash
- **No Data Loss**: Durable persistence with WAL and snapshots
- **Distributed**: Multi-node cluster with leader-based replication
- **Monitoring**: Prometheus metrics for cluster health

### Deployment Modes

```bash
# Standalone development
go run cmd/node/main.go -id=1 -http_addr=:8001 -raft_addr=:9001

# Docker Compose (3-node cluster + monitoring)
docker compose -f docker-compose-monitoring.yml up -d --build

# Access points
Frontend Dashboard:   http://localhost:8080
Node 1 API:          http://localhost:8001
Node 2 API:          http://localhost:8002
Node 3 API:          http://localhost:8003
Prometheus:          http://localhost:9090
Grafana:             http://localhost:3001
```

---

## Quick Start

### 1. Set a Value (Write)

```bash
curl -X POST "http://localhost:8001/set?key=username&value=alice"
# Response: Submitted at index 42 term 3
```

**What happens:**
1. Request reaches Node 1 (assuming it's the leader)
2. Entry is appended to the leader's Raft log
3. Leader replicates the entry to followers
4. Followers acknowledge receipt
5. Leader commits the entry when majority is reached
6. Entry is applied to the state machine

**Consistency Guarantee:** After this returns, all subsequent reads will see `username=alice`.

### 2. Get a Value (Read)

```bash
curl "http://localhost:8001/get?key=username"
# Response: alice
```

**What happens:**
1. Request reaches any node
2. Node returns value from its state machine
3. Value is the latest committed entry

### 3. Check Cluster Health

```bash
curl http://localhost:8001/health
# Response: ok

# Get detailed metrics
curl http://localhost:8001/metrics | head -20
```

---

## API Reference

### Endpoints Summary

| Endpoint | Method | Purpose | Who Can Use |
|----------|--------|---------|-------------|
| `/set` | POST | Store or update a key-value pair | Leader only |
| `/get` | GET | Retrieve a value by key | Any node |
| `/health` | GET | Check if node is alive | Any node |
| `/metrics` | GET | Prometheus metrics | Prometheus/Monitoring |

---

### `POST /set` - Set a Key-Value Pair

**Purpose**: Store or update a value in the distributed store.

**Endpoint**: `POST http://localhost:XXXX/set`

**Request Parameters** (query or JSON body):

| Parameter | Type | Required | Max Length | Description |
|-----------|------|----------|------------|-------------|
| `key` | string | Yes | 256 bytes | The key to store |
| `value` | string | Yes | 1 MB | The value to store |

**Request Examples**:

**Query String Format:**
```bash
curl -X POST "http://localhost:8001/set?key=username&value=alice"
```

**JSON Format:**
```bash
curl -X POST "http://localhost:8001/set" \
  -H "Content-Type: application/json" \
  -d '{"key": "username", "value": "alice"}'
```

**URL-Encoded Form:**
```bash
curl -X POST "http://localhost:8001/set" \
  -d "key=username&value=alice"
```

**Responses**:

| Status | Response | Meaning |
|--------|----------|---------|
| `200` | `Submitted at index 42 term 3` | Value successfully replicated and committed |
| `400` | `Missing key/value` | Required parameter missing |
| `500` | `Not Leader. Leader is node2` | This node is not the leader; redirect to leader |

**CORS Support**: ✅ Yes
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- Preflight OPTIONS requests supported

**Consistency**: ✅ Strong (Linearizable)
- Write is committed only after majority replication
- All subsequent reads will see this value

**Timeout**: Expected < 100ms on healthy cluster (depends on follower latency)

**Leadership Requirement**: ⚠️ **CRITICAL**
- Only the leader accepts SET requests
- Followers will return "Not Leader" error
- The client should:
  1. Remember the leader address
  2. Redirect future SETs to the leader
  3. Fall back to trying other nodes if the leader crashes

---

### `GET /get` - Get a Value by Key

**Purpose**: Retrieve the value associated with a key.

**Endpoint**: `GET http://localhost:XXXX/get?key=<KEY>`

**Request Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `key` | string | Yes | The key to retrieve |

**Request Examples**:

```bash
# Simple GET
curl "http://localhost:8001/get?key=username"
# Response: alice

# From Node 2
curl "http://localhost:8002/get?key=username"
# Response: alice

# Non-existent key
curl "http://localhost:8001/get?key=nonexistent"
# Response: Not found (404)
```

**Responses**:

| Status | Response | Meaning |
|--------|----------|---------|
| `200` | `alice` | Key found; value returned |
| `404` | `Not found` | Key does not exist |
| `400` | `Missing key parameter` | Key parameter not provided |

**CORS Support**: ✅ Yes

**Consistency Guarantee**: ✅ Read sees all committed writes (causal consistency)
- Read reflects the state machine up to the highest committed index
- On leader: Always current
- On follower: Might lag by one replication round (typically < 10ms)

**Data Freshness**:
- **Best**: Read from leader (most current)
- **Acceptable**: Read from any node (might be slightly stale)

---

### `GET /health` - Health Check

**Purpose**: Verify the node is running and responsive.

**Endpoint**: `GET http://localhost:XXXX/health`

**Request Example**:

```bash
curl http://localhost:8001/health
# Response: ok
```

**Response**:

| Status | Response |
|--------|----------|
| `200` | `ok` |

**Note**: This is a simple liveness check. For Raft state (term, leader status, log size), use `/metrics`.

---

### `GET /metrics` - Prometheus Metrics

**Purpose**: Export cluster state and statistics in Prometheus format.

**Endpoint**: `GET http://localhost:XXXX/metrics`

**Response Format**: Prometheus text format

**Key Metrics**:

```
# Raft State
raft_current_term{node_id="1"}              3       # Current term
raft_is_leader{node_id="1"}                 1       # 1 if leader, 0 if follower
raft_log_size{node_id="1"}                  42      # Number of log entries
raft_commit_index{node_id="1"}              40      # Highest committed index
raft_last_applied{node_id="1"}              40      # Highest applied index
raft_last_included_index{node_id="1"}       0       # Last snapshot index

# Replication Progress (leader only)
raft_match_index{node_id="1", peer="2"}    40      # Highest replicated index
raft_next_index{node_id="1", peer="2"}     41      # Next to replicate

# KV Store Statistics
kv_set_total{node_id="1"}                   100     # Total SETs applied
kv_get_total{node_id="1"}                   500     # Total GETs
kv_store_size{node_id="1"}                  42      # Current key count
```

**Scrape Configuration** (for Prometheus):

```yaml
scrape_configs:
  - job_name: 'raft-nodes'
    static_configs:
      - targets:
          - 'node1:8000'
          - 'node2:8000'
          - 'node3:8000'
    scrape_interval: 15s
    scrape_timeout: 5s
```

**Usage with Grafana**:
1. Add Prometheus as data source: `http://prometheus:9090`
2. Create dashboards using the metrics above
3. Pre-built dashboards available in `grafana-dashboards/`

---

## Cluster Architecture

### Leader-Based Model

```
┌─────────────────────────────────────────────────┐
│         Client Requests                         │
└────────────────────┬────────────────────────────┘
                     │
         ┌───────────┼───────────┐
         ▼           ▼           ▼
      Node 1      Node 2      Node 3
      LEADER    FOLLOWER    FOLLOWER
         │           │           │
         └───────────┼───────────┘
                     │
              (Raft Consensus via gRPC)
                     │
              State Machine (Synchronized)
                     │
           [Key-Value Store]
```

**Key Points**:
- **Writes**: Must go to the leader (/set on non-leaders returns error)
- **Reads**: Can go to any node (but followers might be slightly stale)
- **Replication**: Leader sends AppendEntries to followers
- **Consensus**: Entry is committed only after majority acknowledgment

### Node Roles

| Role | Responsibilities |
|------|-----------------|
| **Leader** | Accept writes, replicate to followers, manage heartbeats, decide on commitment |
| **Follower** | Accept reads, apply committed entries, vote in elections |
| **Candidate** | During elections (temporary state) |

### Cluster Sizes

| Size | Max Failures | Min for HA | Notes |
|------|-------------|-----------|-------|
| 1 node | 0 | No | Single point of failure |
| 2 nodes | 0 | No | Requires both nodes up |
| 3 nodes | 1 | ✅ Yes | Recommended minimum |
| 5 nodes | 2 | ✅ Yes | Higher resilience |
| 7 nodes | 3 | ✅ Yes | Enterprise grade |

---

## Consistency Guarantees

### Strong Consistency (Linearizability)

The API provides **strong consistency**, meaning:

1. **Total Order**: All operations have a global order visible to all clients
2. **No Stale Reads**: A read always sees all previously committed writes
3. **Atomic Writes**: Each write either fully succeeds or fully fails (no partial updates)

**Example**:
```
1. Client A: SET key=name, value=alice  → index=42, term=3
2. All nodes: Receive and apply the entry
3. Client B: GET key=name              → Returns "alice"
   (Even if asking a different node)
```

### Consistency Model

- **Write Consistency**: Linearizable (guaranteed by Raft consensus)
- **Read Consistency**: Causal (read sees all committed writes)
- **Cluster Consistency**: Strong (all nodes have the same committed state)

### When Operations Complete

**SET (Write)**:
- Completes when the leader has:
  1. Appended entry to its log
  2. Received acknowledgment from majority of followers
  3. Applied the entry to state machine
- Time: Usually < 100ms (depends on network latency)

**GET (Read)**:
- Completes immediately when queried
- Returns the highest applied state on that node
- On leader: Always current
- On follower: May lag by one replication round (typically < 10ms)

---

## Error Handling

### Common Error Scenarios

#### Scenario 1: Sending Write to Follower

```bash
curl -X POST "http://localhost:8002/set?key=foo&value=bar"
# Response (500): Not Leader. Leader is node1
```

**Solution**:
1. Client should cache the leader address
2. Redirect future writes to the leader
3. If leader crashes, retry other nodes until finding new leader

**Code Example**:
```javascript
async function setValue(key, value) {
  let nodes = ['http://localhost:8001', 'http://localhost:8002', 'http://localhost:8003'];
  
  for (let node of nodes) {
    try {
      const response = await fetch(`${node}/set`, {
        method: 'POST',
        body: JSON.stringify({ key, value })
      });
      
      if (response.ok) {
        return await response.text();  // Success
      }
      
      if (response.status === 500) {
        continue;  // Try next node
      }
    } catch (error) {
      continue;  // Try next node
    }
  }
  
  throw new Error('No leader found');
}
```

#### Scenario 2: Missing Parameters

```bash
curl -X POST "http://localhost:8001/set?key=foo"
# Response (400): Missing key/value
```

**Solution**: Ensure both `key` and `value` are provided.

#### Scenario 3: Key Not Found

```bash
curl "http://localhost:8001/get?key=nonexistent"
# Response (404): Not found
```

**Solution**: Check if the key was previously SET.

#### Scenario 4: Node Crashes

```bash
# Node 1 is down
curl "http://localhost:8001/set?key=foo&value=bar"
# Response: Connection refused
```

**What happens automatically**:
1. Remaining nodes (2, 3) detect Node 1 is down
2. Followers elect new leader (Node 2 or 3)
3. Election completes in ~500ms (timeout-based)
4. New leader accepts writes

**Solution**:
```javascript
// Retry with exponential backoff
async function retrySet(key, value, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await setValue(key, value);
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await new Promise(r => setTimeout(r, 100 * Math.pow(2, i)));
    }
  }
}
```

---

## Examples

### Example 1: Setting and Getting Values

**Shell Script**:
```bash
#!/bin/bash

LEADER="http://localhost:8001"

# Set multiple values
echo "Setting values..."
curl -X POST "$LEADER/set?key=name&value=alice"
curl -X POST "$LEADER/set?key=age&value=30"
curl -X POST "$LEADER/set?key=city&value=seattle"

# Get values
echo "Getting values..."
curl "$LEADER/get?key=name"    # alice
curl "$LEADER/get?key=age"     # 30
curl "$LEADER/get?key=city"    # seattle
```

### Example 2: Client-Side Leader Detection

**JavaScript**:
```javascript
class RaftKVClient {
  constructor(nodes) {
    this.nodes = nodes;
    this.leaderIndex = 0;  // Start with first node
  }

  async set(key, value) {
    for (let i = 0; i < this.nodes.length; i++) {
      const node = this.nodes[(this.leaderIndex + i) % this.nodes.length];
      
      try {
        const response = await fetch(`${node}/set`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ key, value })
        });

        if (response.ok) {
          this.leaderIndex = (this.leaderIndex + i) % this.nodes.length;
          return await response.text();
        }
      } catch (error) {
        console.error(`Error on ${node}:`, error);
      }
    }

    throw new Error('All nodes failed');
  }

  async get(key) {
    const node = this.nodes[this.leaderIndex];
    const response = await fetch(`${node}/get?key=${key}`);
    
    if (response.ok) {
      return await response.text();
    } else if (response.status === 404) {
      return null;
    }
    
    throw new Error(`GET failed: ${response.statusText}`);
  }
}

// Usage
const client = new RaftKVClient([
  'http://localhost:8001',
  'http://localhost:8002',
  'http://localhost:8003'
]);

await client.set('username', 'alice');
const value = await client.get('username');
console.log(value);  // alice
```

### Example 3: Bulk Operations

**Python**:
```python
import requests
import time

NODES = [
    'http://localhost:8001',
    'http://localhost:8002',
    'http://localhost:8003'
]

class RaftKVStore:
    def __init__(self, nodes):
        self.nodes = nodes
        self.leader_idx = 0
    
    def _get_leader(self):
        return self.nodes[self.leader_idx % len(self.nodes)]
    
    def set(self, key, value):
        for attempt in range(len(self.nodes)):
            node = self._get_leader()
            try:
                resp = requests.post(
                    f'{node}/set',
                    params={'key': key, 'value': value},
                    timeout=5
                )
                if resp.status_code == 200:
                    return resp.text
                elif resp.status_code == 500:
                    self.leader_idx += 1
            except requests.RequestException:
                self.leader_idx += 1
        
        raise Exception('Set failed on all nodes')
    
    def get(self, key):
        for node in self.nodes:
            try:
                resp = requests.get(
                    f'{node}/get',
                    params={'key': key},
                    timeout=5
                )
                if resp.status_code == 200:
                    return resp.text
            except requests.RequestException:
                continue
        
        raise Exception('Get failed on all nodes')

# Usage
store = RaftKVStore(NODES)

# Bulk write
for i in range(100):
    store.set(f'key-{i}', f'value-{i}')
    print(f'Stored key-{i}')

# Bulk read
for i in range(100):
    value = store.get(f'key-{i}')
    assert value == f'value-{i}'
    print(f'key-{i} = {value}')
```

---

## Monitoring

### Prometheus Integration

**Scrape Configuration**:
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'raft-nodes'
    static_configs:
      - targets:
          - 'node1:8000'
          - 'node2:8000'
          - 'node3:8000'
```

### Grafana Dashboards

Pre-built dashboards included:
- **Cluster Status**: Leader, term, log size
- **Replication Health**: Match/next indices for each peer
- **State Machine**: Applied index, store size
- **Performance**: Operation latency, throughput

### Key Metrics to Monitor

| Metric | Alert Threshold | Meaning |
|--------|-----------------|---------|
| `raft_is_leader` | Should be 1 on exactly one node | Leader detection |
| `raft_log_size` | Growing linearly | Entries being appended |
| `raft_commit_index` | Should equal `raft_log_size` | Entries being committed |
| `raft_last_applied` | Should equal `raft_commit_index` | Entries being applied |
| `raft_current_term` | Increasing only during elections | Normal term progression |

---

## Troubleshooting

### Problem: "Not Leader. Leader is unknown"

**Cause**: Cluster is in election (no leader elected yet)

**Solution**:
```bash
# Wait for leader election (~500ms)
sleep 1

# Retry the request
curl -X POST "http://localhost:8001/set?key=foo&value=bar"
```

### Problem: CORS Errors in Browser

**Cause**: Browser blocks cross-origin request

**Verify**: Check response headers:
```bash
curl -i -X OPTIONS http://localhost:8001/set | grep Access-Control
```

**Solution**: Already implemented in backend (all endpoints have CORS headers)

### Problem: Slow Writes (> 500ms)

**Cause**: Network latency or slow followers

**Check**:
```bash
curl http://localhost:8001/metrics | grep raft_
```

**Solutions**:
1. Verify network latency: `ping node2`, `ping node3`
2. Check if followers are healthy: `curl http://localhost:8002/health`
3. Monitor log replication: Check `raft_match_index` metrics

### Problem: Data Not Found After Write

**Cause**: Wrote to follower or uncommitted entry

**Solution**:
```bash
# Verify you wrote to leader
curl -X POST "http://localhost:8001/set?key=test&value=123"

# Wait for commitment (usually instant)
sleep 1

# Read from any node
curl "http://localhost:8002/get?key=test"
```

### Problem: Metric Endpoint Returns 404

**Cause**: Metrics handler not registered

**Solution**: Ensure `/metrics` is exposed in Docker compose:
```yaml
- "9090:8000"  # Prometheus scrapes on 8000
```

---

## Best Practices

### 1. Cache the Leader Address

```javascript
let cachedLeader = 'http://localhost:8001';

async function set(key, value) {
  try {
    return await fetch(`${cachedLeader}/set`, {...});
  } catch {
    // Clear cache and retry
    cachedLeader = null;
    return await retryAllNodes(key, value);
  }
}
```

### 2. Implement Retry Logic

```javascript
async function retryWithBackoff(fn, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await sleep(100 * Math.pow(2, i));
    }
  }
}
```

### 3. Read from Leader for Consistency

```javascript
async function readStrict(key) {
  // Always read from known leader
  return await fetch(`${leaderAddress}/get?key=${key}`);
}
```

### 4. Monitor Cluster Health

```javascript
async function healthCheck() {
  for (let node of NODES) {
    try {
      await fetch(`${node}/health`, { timeout: 5000 });
      console.log(`${node}: UP`);
    } catch {
      console.log(`${node}: DOWN`);
    }
  }
}
```

---

## References

- [Raft Consensus Algorithm](https://raft.io/raft.pdf)
- [Prometheus Metrics](https://prometheus.io/docs/introduction/overview/)
- [Grafana Dashboards](https://grafana.com/grafana/dashboards/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)

---

**Last Updated**: 2024
**API Version**: 1.0.0
**Status**: Production Ready
