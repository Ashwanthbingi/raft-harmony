# Raftline Frontend - Raft Backend Integration Guide

## Overview

This document describes the integration of the React/TypeScript frontend with the Go-based Raft backend cluster.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    React Frontend (Raftline)                    │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ src/lib/api.ts                                           │   │
│  │ - RaftAPI client                                         │   │
│  │ - HTTP communication with backend nodes                  │   │
│  │ - SET/GET operations                                     │   │
│  │ - Cluster status monitoring                              │   │
│  └──────────────────────────────────────────────────────────┘   │
│                            │                                     │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Components:                                              │   │
│  │ - KVOperations: SET/GET UI                              │   │
│  │ - ClusterStatus: Node monitoring                         │   │
│  │ - Operations Route: Full console                         │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                            │
                  HTTP REST API (JSON)
                            │
┌─────────────────────────────────────────────────────────────────┐
│                     Raft Cluster (Go)                           │
│  ┌──────────────────┐  ┌──────────────────┐ ┌──────────────────┐
│  │  Node 1 (Leader) │  │  Node 2 (Foll.)  │ │ Node 3 (Foll.)   │
│  │ http://...:8001  │  │ http://...:8002  │ │ http://...:8003  │
│  │ - POST /set      │  │ - POST /set      │ │ - POST /set      │
│  │ - GET /get       │  │ - GET /get       │ │ - GET /get       │
│  └──────────────────┘  └──────────────────┘ └──────────────────┘
│         │ gRPC ↔ │        │ gRPC ↔ │        │ gRPC ↔ │
│         └────────────────────────────────────────────┘
│                    Raft Consensus
└─────────────────────────────────────────────────────────────────┘
```

## File Structure

```
src/
├── lib/
│   └── api.ts                    # RaftAPI client service
├── components/
│   ├── kv-operations.tsx        # SET/GET form component
│   ├── cluster-status.tsx       # Cluster monitoring component
│   └── site-nav.tsx             # Updated navigation
├── routes/
│   ├── dashboard.tsx            # Updated with KV operations
│   └── operations.tsx           # New full operations console
```

## Key Components

### 1. **RaftAPI Client** (`src/lib/api.ts`)

The main interface for communicating with the backend:

```typescript
// Set a key-value pair
const response = await raftAPI.set("key", "value");

// Get a value
const response = await raftAPI.get("key");

// Check cluster status
const status = await raftAPI.getClusterStatus();

// Update node configuration
raftAPI.updateNodes(newNodesList);
```

**Features:**
- Automatic leader detection and failover
- HTTP client interface (no gRPC on frontend)
- Error handling and retry logic
- Node health checking

### 2. **KV Operations Component** (`src/components/kv-operations.tsx`)

A dual-panel interface for:
- **Left Panel**: SET operation (submit key-value pairs)
- **Right Panel**: GET operation (retrieve values)

**Features:**
- Form validation
- Loading states
- Success/error feedback
- Copy-to-clipboard for GET results

### 3. **Cluster Status Component** (`src/components/cluster-status.tsx`)

Real-time cluster monitoring displaying:
- Node list with status (Leader/Follower/Candidate)
- Health indicators
- Term information
- Auto-refresh every 3 seconds

### 4. **Operations Route** (`src/routes/operations.tsx`)

A dedicated console page featuring:
- Full cluster status display
- KV operations with more space
- Getting started guide
- Connection troubleshooting

## Integration Points

### Backend API Expectations

The frontend expects the backend to provide:

#### SET Operation
```
POST /set
Content-Type: application/json

{
  "key": "string",
  "value": "string"
}

Response (success):
200 OK
Submitted at index 1042 term 42

Response (not leader):
200 OK
Not Leader. Leader is Unknown
```

#### GET Operation
```
GET /get?key=<key>

Response (found):
200 OK
<value>

Response (not found):
404 Not Found
```

### Default Node Configuration

```typescript
const DEFAULT_NODES = [
  {
    id: 'node1',
    address: 'localhost',
    httpPort: 8001,
    isLeader: true,
  },
  {
    id: 'node2',
    address: 'localhost',
    httpPort: 8002,
    isLeader: false,
  },
  {
    id: 'node3',
    address: 'localhost',
    httpPort: 8003,
    isLeader: false,
  },
];
```

## Running the Application

### 1. **Start the Backend Raft Cluster**

```bash
cd /path/to/distributed-kv-raft-main

# Terminal 1: Start Node 2 (Follower)
go run cmd/node/main.go \
  --id=node2 \
  --http_addr=127.0.0.1:8002 \
  --raft_addr=127.0.0.1:9002 \
  --peers=127.0.0.1:9001,127.0.0.1:9003

# Terminal 2: Start Node 3 (Follower)
go run cmd/node/main.go \
  --id=node3 \
  --http_addr=127.0.0.1:8003 \
  --raft_addr=127.0.0.1:9003 \
  --peers=127.0.0.1:9001,127.0.0.1:9002

# Terminal 3: Start Node 1 (Leader)
go run cmd/node/main.go \
  --id=node1 \
  --http_addr=127.0.0.1:8001 \
  --raft_addr=127.0.0.1:9001 \
  --peers=127.0.0.1:9002,127.0.0.1:9003
```

### 2. **Start the Frontend**

```bash
cd /path/to/raft-harmony

# Install dependencies
npm install
# or
bun install

# Run dev server
npm run dev
# or
bun dev
```

The frontend will be available at: `http://localhost:5173` (or similar)

### 3. **Access the Interfaces**

- **Dashboard**: `http://localhost:5173/dashboard` - Full monitoring with quick ops
- **Operations Console**: `http://localhost:5173/operations` - Full KV operations
- **Home**: `http://localhost:5173/` - Landing page

## Usage Workflow

### Testing SET Operation

1. Navigate to **Operations** page
2. Fill in Key field: `greeting`
3. Fill in Value field: `hello from raft`
4. Click **Set Value**
5. Watch the success notification

### Testing GET Operation

1. In the **Get Value** panel
2. Fill in Key field: `greeting`
3. Click **Get Value**
4. View the returned value: `hello from raft`

### Monitoring Cluster Status

The **Cluster Status** component updates every 3 seconds and shows:
- Node IDs and endpoints
- Current state (Leader/Follower/Candidate)
- Current term
- Health indicators (animated bars)

## Error Handling

### Common Errors and Solutions

| Error | Cause | Solution |
|-------|-------|----------|
| Connection refused | Backend not running | Start Raft cluster with commands above |
| Not Leader error | Sent to follower node | Use `/set` only on leader or auto-redirect |
| Empty response | Backend crashed | Restart nodes, check logs |
| 404 on GET | Key doesn't exist | Verify SET operation succeeded first |

## Future Enhancements

Based on the PDF requirements, these features can be added:

1. **Persistent Storage Integration** - Display persistence status in UI
2. **Snapshot Monitoring** - Show snapshot creation/installation events
3. **Metrics Dashboard** - Integrate Prometheus metrics display
4. **Real-time Logs** - Stream logs from backend to UI
5. **Failover Testing** - UI controls to simulate node failures
6. **Performance Charts** - Latency/throughput graphs

## Architecture Diagram

```mermaid
graph TB
    Client["👤 Client (Browser)"]
    
    Client -->|HTTP| FE["🎨 React Frontend"]
    
    FE -->|GET /get?key=x| N1["🖥️ Node 1<br/>(Leader)<br/>:8001"]
    FE -->|POST /set| N1
    FE -->|GET /get?key=x| N2["🖥️ Node 2<br/>(Follower)<br/>:8002"]
    FE -->|GET /get?key=x| N3["🖥️ Node 3<br/>(Follower)<br/>:8003"]
    
    N1 ←→|gRPC<br/>Append/Vote| N2
    N1 ←→|gRPC<br/>Append/Vote| N3
    N2 ←→|gRPC<br/>Append/Vote| N3
    
    N1 --> KV1["Key-Value<br/>Store"]
    N2 --> KV2["Key-Value<br/>Store"]
    N3 --> KV3["Key-Value<br/>Store"]
```

## Testing Checklist

- [ ] Start all 3 nodes successfully
- [ ] See all nodes in Cluster Status
- [ ] Node 1 shows as Leader
- [ ] Can SET a value through UI
- [ ] Can GET the value back
- [ ] Cluster Status shows correct term
- [ ] Try GET from a follower node
- [ ] Cluster Status auto-refreshes
- [ ] Copy button works in GET result

## Troubleshooting

### Frontend won't connect to backend

**Check:**
1. Backend nodes are running on correct ports (8001-8003)
2. `api.ts` DEFAULT_NODES matches your node configuration
3. No firewall blocking localhost traffic
4. Browser console shows any CORS or network errors

**Fix:**
```typescript
// In src/lib/api.ts, update DEFAULT_NODES if needed
export const DEFAULT_NODES: ClusterNode[] = [
  {
    id: 'node1',
    address: 'your-host',  // Change if needed
    httpPort: 8001,
    // ...
  },
];
```

### Cluster shows as "down" in UI

**Check:**
1. Try manual curl to verify backend:
   ```bash
   curl http://localhost:8001/get?key=test
   ```
2. Backend logs for errors
3. Node processes still running

### Values not persisting between requests

**Note:** The current backend is in-memory only (no persistence).
- Values only exist in the current process
- Restarting a node clears all data
- This is being addressed in Task 2 (Persistent Storage)

## Contact & Support

For issues with:
- **Backend Raft logic**: Check `/path/to/distributed-kv-raft-main/README.md`
- **Frontend integration**: Review this file
- **General architecture**: See PDF Executive Summary
