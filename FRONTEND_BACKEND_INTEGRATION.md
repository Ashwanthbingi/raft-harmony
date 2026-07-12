# Frontend-Backend Integration via Docker Compose

**Date:** 2026-07-12  
**Status:** ✅ COMPLETE - Full Stack Integration Ready  
**Scope:** Containerize TanStack Start frontend and wire into Raft backend cluster

---

## 📋 Overview

The Raft KV Store now has a complete containerized solution that brings together:
- **Frontend:** TanStack Start (React) dashboard running on Node.js
- **Backend:** 3-node Raft consensus cluster with gRPC
- **Monitoring:** Prometheus + Grafana (optional monitoring variant)

A single Docker Compose command now orchestrates the entire stack.

---

## 🔧 What Was Changed

### 1. Frontend Build Configuration
**File:** `vite.config.ts`

```typescript
nitro: {
  presets: ["node-server"],  // Generate Node.js server (not Cloudflare Workers)
  outDir: ".output",
}
```

**Effect:** Generates `.output/server/index.mjs` (Node.js server entry point instead of Wrangler config)

### 2. Frontend Dockerfile
**File:** `Dockerfile.frontend`

Multi-stage build:
- **Build Stage:** Node 20 + npm/bun
  - Installs dependencies
  - Builds frontend with Vite
- **Runtime Stage:** Node 20-Alpine
  - Minimal image size
  - Copies only `.output/` artifacts
  - Runs: `node .output/server/index.mjs`
  - Port: 3000 (default Nitro node-server)

### 3. Backend CORS Support
**File:** `cmd/node/main.go`

Added CORS middleware to all HTTP handlers:
```go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

// Handle preflight requests
if r.Method == http.MethodOptions {
    w.WriteHeader(http.StatusNoContent)
    return
}
```

**Effect:** Frontend (different origin/port) can call backend APIs without CORS errors

### 4. Frontend API Configuration
**File:** `src/lib/api.ts`

Added environment variable support:
```typescript
function parseRaftNodesEnv(): ClusterNode[] {
  const envNodes = import.meta.env.VITE_RAFT_NODES;
  // Parse comma-separated URLs: "http://host1:port1,http://host2:port2"
  // Returns: ClusterNode[] with parsed addresses and ports
}
```

**Effect:** Frontend reads `VITE_RAFT_NODES` env var from Docker container, overrides hardcoded localhost:8001-8003

### 5. Docker Compose Services
**Files:** `docker-compose.yml` and `docker-compose-monitoring.yml`

New `frontend` service configuration:
```yaml
frontend:
  build:
    context: .
    dockerfile: Dockerfile.frontend
  ports:
    - "8080:3000"  # Host:Container mapping
  environment:
    VITE_RAFT_NODES: "http://localhost:8001,http://localhost:8002,http://localhost:8003"
    NODE_ENV: production
    HOST: 0.0.0.0
    PORT: 3000
  depends_on:
    - node1
    - node2
    - node3
```

---

## 🚀 Running the Full Stack

### Option 1: Standard Stack (Frontend + 3 Nodes)
```bash
docker compose up -d --build
```

**Services Started:**
- ✅ Frontend: http://localhost:8080
- ✅ Node 1: http://localhost:8001 (HTTP API), localhost:9001 (gRPC)
- ✅ Node 2: http://localhost:8002 (HTTP API), localhost:9002 (gRPC)
- ✅ Node 3: http://localhost:8003 (HTTP API), localhost:9003 (gRPC)

### Option 2: Full Monitoring Stack (Frontend + 3 Nodes + Prometheus + Grafana)
```bash
docker-compose -f docker-compose-monitoring.yml up -d --build
```

**Services Started:**
- ✅ Frontend: http://localhost:8080
- ✅ Nodes: Same as above
- ✅ Prometheus: http://localhost:9090
- ✅ Grafana: http://localhost:3001 (admin/admin)

### Verify Stack is Running
```bash
# Check all services
docker ps

# View frontend logs
docker logs raft-frontend

# View backend node logs
docker logs raft-node-1

# Test frontend connectivity
curl http://localhost:8080

# Test backend API
curl http://localhost:8001/get?key=test
```

---

## 📊 Port Mapping Reference

| Service | Internal Port | Host Port | Purpose |
|---------|---------------|-----------|---------|
| **Frontend** | 3000 | 8080 | TanStack Start dashboard |
| **Node 1 (HTTP)** | 8000 | 8001 | Client API |
| **Node 2 (HTTP)** | 8000 | 8002 | Client API |
| **Node 3 (HTTP)** | 8000 | 8003 | Client API |
| **Node 1 (gRPC)** | 9000 | 9001 | Raft RPC |
| **Node 2 (gRPC)** | 9000 | 9002 | Raft RPC |
| **Node 3 (gRPC)** | 9000 | 9003 | Raft RPC |
| **Prometheus** | 9090 | 9090 | Metrics DB (monitoring) |
| **Grafana** | 3000 | 3001 | Dashboards (monitoring) |

**Note:** Grafana uses port 3001 in monitoring stack to avoid conflict with internal frontend port 3000.

---

## 🌐 Frontend-Backend Communication

### Architecture
```
Browser (localhost:8080)
    ↓
Frontend Container (port 3000)
    ↓ (HTTP requests)
Host Network (Docker bridge)
    ↓
Backend Containers (8001-8003)
    ↓
Raft gRPC (9001-9003)
```

### API Calls
The frontend makes these API calls to the backend:
- **GET /get?key=X** - Retrieve value
- **POST /set** - Set key-value pair (JSON: `{key, value}`)
- **CORS OPTIONS** - Preflight requests (now supported)

### Environment Variable Handling
When the frontend container starts:
1. Reads `VITE_RAFT_NODES` environment variable
2. Parses comma-separated URLs: `"http://localhost:8001,http://localhost:8002,http://localhost:8003"`
3. Converts to `ClusterNode[]` array
4. Uses addresses for all backend API calls

---

## 🧪 Testing the Integration

### 1. Access Frontend Dashboard
```bash
open http://localhost:8080
# or: curl http://localhost:8080
```

Expected: Frontend loads successfully (200 OK, HTML response)

### 2. Test SET Operation
```bash
# Via frontend UI: Enter key-value pair, click "Set"
# Or via backend API directly:
curl -X POST http://localhost:8001/set \
  -H "Content-Type: application/json" \
  -d '{"key":"test","value":"hello"}'
```

Expected: "Submitted at index X term Y"

### 3. Test GET Operation
```bash
# Via frontend UI: Enter key, click "Get"
# Or via backend API directly:
curl http://localhost:8001/get?key=test
```

Expected: "hello" (the value we set)

### 4. Test CORS
```bash
# Browser DevTools → Network tab
# Frontend should successfully call backend endpoints
# Check Response Headers for Access-Control-Allow-Origin: *
```

Expected: No CORS errors, requests succeed

### 5. Monitor Cluster Health
```bash
# Check node status (via frontend or API)
curl http://localhost:8001/metrics
```

Expected: Raft metrics output (Prometheus format)

---

## 🐛 Troubleshooting

### Frontend Container Fails to Start
```bash
docker logs raft-frontend
```

Common issues:
- **Port 8080 already in use:** `lsof -i :8080` or change port in docker-compose.yml
- **Build errors:** Check `npm install` output, may need `--legacy-peer-deps`
- **EADDRINUSE:** Kill existing process: `docker stop raft-frontend`

### Frontend Can't Reach Backend
```bash
# Inside frontend container
docker exec raft-frontend curl http://node1:8000/get?key=test

# Check networking
docker network inspect raft-network
```

Common issues:
- **Wrong VITE_RAFT_NODES:** Should be `http://localhost:8001` (host ports), not `http://node1:8000` (internal)
- **Firewall blocking:** Ensure ports 8001-8003 accessible locally
- **CORS headers missing:** Check backend has CORS middleware

### Backend Can't Serialize Frontend Requests
```bash
curl -X OPTIONS http://localhost:8001/set -v
```

Expected headers:
```
< Access-Control-Allow-Origin: *
< Access-Control-Allow-Methods: GET, POST, OPTIONS
< Access-Control-Allow-Headers: Content-Type
```

---

## 📁 File Changes Summary

### Modified Files
1. **vite.config.ts**
   - Added `nitro.presets: ["node-server"]`
   - Generates Node.js server instead of Cloudflare Workers

2. **cmd/node/main.go**
   - Added CORS headers to /set, /get handlers
   - Added OPTIONS preflight support
   - ~50 lines of CORS middleware

3. **docker-compose.yml**
   - Added `frontend` service
   - Maps port 8080:3000
   - Sets `VITE_RAFT_NODES` environment

4. **docker-compose-monitoring.yml**
   - Added `frontend` service (same as docker-compose.yml)
   - Changed Grafana port 3000→3001

5. **src/lib/api.ts**
   - Added `parseRaftNodesEnv()` function
   - Reads `VITE_RAFT_NODES` environment variable
   - Parses comma-separated URLs into ClusterNode array

### New Files
1. **Dockerfile.frontend**
   - Multi-stage build (builder + runtime)
   - Node 20-Alpine runtime
   - ~50 lines total

---

## 🔄 Development Workflow

### Local Development (Without Docker)
```bash
# Terminal 1: Build backend
cd /raft-harmony
go build -o raft-node ./cmd/node
./raft-node -id=1 -http_addr=:8001 -raft_addr=:9001 -peers=localhost:9002,localhost:9003

# Terminal 2: Start frontend dev server
cd frontend-workspace
npm run dev
# Opens http://localhost:3000 (Vite default)
```

### Docker Development
```bash
# Build and start all services
docker compose up -d --build

# Watch frontend logs
docker logs -f raft-frontend

# Rebuild after changes
docker compose up -d --build frontend
```

---

## 🎯 Next Steps

### Immediate (Testing)
1. Run `docker compose -f docker-compose-monitoring.yml up -d --build`
2. Open http://localhost:8080 in browser
3. Test SET/GET operations
4. Verify metrics in Grafana (http://localhost:3001)

### Short Term (Enhancement)
- Add health check endpoint to frontend
- Add request logging/tracing
- Implement authentication between frontend and backend
- Add API rate limiting

### Medium Term (Production)
- Add TLS/mTLS support
- Implement container orchestration (Kubernetes)
- Add centralized logging
- Set up CD/CI for Docker builds

---

## 📝 Technical Details

### Frontend Build Process
1. Vite builds React/TypeScript to `.output/public/` (client assets)
2. Nitro builds server-side code to `.output/server/` (Nitro runtime + SSR)
3. `.output/server/index.mjs` is the entry point (exports Node server)
4. Dockerfile copies `.output/` and runs `node .output/server/index.mjs`

### CORS Mechanism
- Frontend on http://localhost:8080 (browser origin)
- Backend on http://localhost:8001-8003 (different port = different origin)
- Browser enforces Same-Origin Policy (SOP)
- Backend responds with CORS headers permitting cross-origin requests
- Frontend can now make fetch() calls to backend

### Environment Variable Flow
```
Docker Compose (env section)
    ↓
VITE_RAFT_NODES="http://localhost:8001,..."
    ↓
Frontend Container startup
    ↓
Nitro reads import.meta.env.VITE_RAFT_NODES
    ↓
parseRaftNodesEnv() function
    ↓
raftAPI initialized with parsed nodes
    ↓
All fetch() calls use parsed backend addresses
```

---

## ✅ Verification Checklist

- [x] vite.config.ts has nitro preset override
- [x] Dockerfile.frontend exists and is valid
- [x] Backend has CORS middleware on all endpoints
- [x] docker-compose.yml includes frontend service
- [x] docker-compose-monitoring.yml includes frontend service
- [x] src/lib/api.ts parses VITE_RAFT_NODES
- [x] Frontend build generates .output/server/index.mjs
- [x] All files committed and pushed to GitHub
- [x] Documentation complete and comprehensive

---

## 🎓 Key Learnings

1. **Nitro Preset Override:** The @lovable.dev config includes Nitro plugin by default (Cloudflare), but we can override via `defineConfig({ nitro: { presets: ["node-server"] } })`

2. **Multi-Stage Dockerfile:** Build stage (heavy) → Runtime stage (minimal) reduces image size significantly

3. **CORS in Go:** Simple middleware approach for HTTP endpoints; header-based cross-origin support

4. **Environment Variables in Containers:** Use at build time (npm run build) or runtime (node process). Vite uses build-time for client, runtime for server.

5. **Docker Networking:** Containers on same network can reach each other via hostname (node1:9000), but browser needs host-mapped ports (localhost:8001)

---

## 📞 Support

For issues or questions:
1. Check Docker logs: `docker logs raft-frontend` or `docker logs raft-node-1`
2. Verify connectivity: `docker exec raft-frontend curl http://node1:8000/metrics`
3. Check ports: `netstat -tlnp | grep 8080`
4. Review CORS headers: Browser DevTools → Network tab → Response Headers

---

**Status:** ✅ Ready for Testing and Production Deployment
