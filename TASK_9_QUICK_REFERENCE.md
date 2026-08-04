# Quick Reference: Task 9 Advanced API Features

## 🚀 New Endpoints Summary

### Range Queries

#### 1. Get Keys in Range
```bash
curl "http://localhost:8001/range?start=user:100&end=user:200"
```
Returns all keys between `user:100` and `user:200` (exclusive end)

#### 2. Get Keys by Prefix
```bash
curl "http://localhost:8001/prefix?prefix=config:"
```
Returns all keys starting with `config:`

#### 3. List All Keys (Paginated)
```bash
curl "http://localhost:8001/keys?limit=50&offset=0"
curl "http://localhost:8001/keys?limit=50&offset=50"  # Next page
```
Returns sorted keys with pagination support

---

### Batch Operations

#### 4. Batch GET (Multiple Keys)
```bash
curl -X POST http://localhost:8001/batch/get \
  -H "Content-Type: application/json" \
  -d '{"keys": ["key1", "key2", "key3"]}'
```
Retrieves values for multiple keys in one request

#### 5. Batch SET (Multiple Keys)
```bash
curl -X POST http://localhost:8001/batch/set \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {"key": "key1", "value": "value1"},
      {"key": "key2", "value": "value2"}
    ]
  }'
```
Sets multiple key-value pairs atomically

#### 6. Batch DELETE (Multiple Keys)
```bash
curl -X POST http://localhost:8001/batch/delete \
  -H "Content-Type: application/json" \
  -d '{"keys": ["key1", "key2", "key3"]}'
```
Deletes multiple keys atomically

---

### Health & Status

#### 7. Enhanced Health Check
```bash
curl "http://localhost:8001/health" | jq .
```
Returns cluster health with detailed metrics:
```json
{
  "status": "healthy",
  "isLeader": true,
  "role": "leader",
  "currentTerm": 5,
  "logSize": 1250,
  "commitIndex": 1240,
  "lastApplied": 1240,
  "storeSize": 500,
  "timestamp": 1720028400
}
```

---

## 📊 Use Case Examples

### Example 1: Store Multiple Settings
```bash
curl -X POST http://localhost:8001/batch/set \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {"key": "app:version", "value": "1.2.3"},
      {"key": "app:port", "value": "8080"},
      {"key": "app:timeout", "value": "30000"},
      {"key": "app:maxConnections", "value": "1000"}
    ]
  }'
```

### Example 2: Retrieve All User Data
```bash
# Get all keys for a specific user
curl "http://localhost:8001/prefix?prefix=user:123:"

# Or batch get specific fields
curl -X POST http://localhost:8001/batch/get \
  -H "Content-Type: application/json" \
  -d '{
    "keys": [
      "user:123:name",
      "user:123:email",
      "user:123:role",
      "user:123:status"
    ]
  }'
```

### Example 3: Browse Leaderboard
```bash
# Get first 100 entries
curl "http://localhost:8001/keys?limit=100&offset=0"

# Get next page
curl "http://localhost:8001/keys?limit=100&offset=100"

# Get scores in range
curl "http://localhost:8001/range?start=score:1000&end=score:2000"
```

### Example 4: Time Series Queries
```bash
# Get all metrics from an hour
curl "http://localhost:8001/range?start=metric:2024-07-10T12:00:00&end=metric:2024-07-10T13:00:00"

# Get all temperature readings
curl "http://localhost:8001/prefix?prefix=sensor:temperature:"
```

---

## ⚡ Performance Tips

### For Large Result Sets
- Use pagination with `/keys?limit=N&offset=M`
- Don't fetch entire dataset at once
- Process results in pages

### For Batch Operations
- Batch 50-500 operations together for best throughput
- Each operation is logged individually (no partial rollback)
- Expect ~50-200ms per 100 operations

### For Range Queries
- Keep ranges moderate (< 10K keys)
- Use prefix for hierarchical organization
- Pagination available for large ranges

---

## 🛠️ Testing Commands

```bash
# 1. Create test data
for i in {1..100}; do
  curl -X POST "http://localhost:8001/set?key=item:$i&value=value$i"
done

# 2. Test range query
curl "http://localhost:8001/range?start=item:10&end=item:20" | jq '.count'

# 3. Test prefix query
curl "http://localhost:8001/prefix?prefix=item:1" | jq '.count'

# 4. Test batch operations
curl -X POST http://localhost:8001/batch/set \
  -H "Content-Type: application/json" \
  -d '{
    "operations": [
      {"key": "test:1", "value": "a"},
      {"key": "test:2", "value": "b"},
      {"key": "test:3", "value": "c"}
    ]
  }' | jq '.success'

# 5. Check health
curl "http://localhost:8001/health" | jq '.isLeader'
```

---

## 📝 Response Formats

### Range Query Response
```json
{
  "keys": {"key1": "value1", "key2": "value2"},
  "count": 2,
  "truncated": false,
  "range": {"start": "key1", "end": "key3"}
}
```

### Batch Operation Response
```json
{
  "results": {
    "key1": {"success": true, "index": 100, "term": 5},
    "key2": {"success": true, "index": 101, "term": 5}
  },
  "total": 2,
  "success": 2,
  "failed": 0
}
```

### List Keys Response
```json
{
  "keys": ["key1", "key2", "key3"],
  "count": 3,
  "total": 5000,
  "offset": 0,
  "limit": 100,
  "truncated": true,
  "hasMore": true
}
```

---

## 🔍 Common Queries

| Use Case | Endpoint | Example |
|----------|----------|---------|
| Get one value | `/get?key=X` | `/get?key=user:1` |
| Set one value | `/set?key=X&value=Y` | `/set?key=user:1&value=Alice` |
| Get many values | `/batch/get` | POST with list of keys |
| Set many values | `/batch/set` | POST with operations |
| Delete many | `/batch/delete` | POST with list of keys |
| Range scan | `/range` | `/range?start=a&end=z` |
| Prefix scan | `/prefix` | `/prefix?prefix=user:` |
| List all | `/keys` | `/keys?limit=100` |
| Check health | `/health` | `/health` |

---

## ✅ All Features at a Glance

- ✅ Range queries by lexicographic range
- ✅ Prefix-based key queries
- ✅ Paginated key listing
- ✅ Batch GET operations
- ✅ Batch SET operations
- ✅ Batch DELETE operations
- ✅ Enhanced health monitoring
- ✅ Full CORS support
- ✅ Comprehensive error handling
- ✅ Production-ready APIs

---

**For detailed documentation, see: `ADVANCED_API_FEATURES.md`**
