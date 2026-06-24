---
comet_change: node-registry-dashboard
role: technical-design
canonical_spec: openspec
---

# Node Registry Dashboard — Technical Design

## 1. Redis Data Model

```
Key: dagflow:node:<nodeId>
Value: JSON {nodeId, address, functions[], lastHeartbeat (unix ts)}
TTL: 5 minutes (nodeTTL)

Key: dagflow:func:<funcName>
Value: SET {nodeId1, nodeId2, ...}
No explicit TTL — members are filtered by node key existence at query time

Constants:
  heartbeatThreshold = 30s   — window for "online" status
  nodeTTL            = 5min   — cleanup window for dead nodes
```

## 2. API Design

### 2.1 Proto extensions (`remote_executor.proto`)

```protobuf
service NodeRegistry {
  // existing
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  // new
  rpc ListNodes(ListNodesRequest) returns (ListNodesResponse);
  rpc GetNode(GetNodeRequest) returns (GetNodeResponse);
}

message NodeDetail {
  string node_id = 1;
  string address = 2;
  repeated string functions = 3;
  string status = 4;         // "online" | "offline"
  int64 last_heartbeat = 5;
}

message ListNodesRequest {}
message ListNodesResponse { repeated NodeDetail items = 1; int32 total = 2; }
message GetNodeRequest { string node_id = 1; }
message GetNodeResponse { NodeDetail node = 1; }
```

### 2.2 HTTP routes

```
GET /api/v1/nodes       → ListNodes (via gRPC handler bridge)
GET /api/v1/nodes/:id   → GetNode  (via gRPC handler bridge)
```

### 2.3 Status derivation

```
status = (now - lastHeartbeat) < heartbeatThreshold ? "online" : "offline"
```

## 3. Implementation Details

### 3.1 Register (refactored)

1. Marshal `NodeInfo{NodeID, Address, Functions, LastHeartbeat: now}` → JSON
2. `SET dagflow:node:<nodeId> <json> EX nodeTTL`
3. For each fn in Functions: `SADD dagflow:func:<fn> <nodeId>`

### 3.2 Heartbeat (refactored)

1. `GET dagflow:node:<nodeId>` → unmarshal
2. Update `lastHeartbeat = now`
3. `SET dagflow:node:<nodeId> <json> EX nodeTTL`

### 3.3 ListNodes (new)

1. `SCAN 0 match dagflow:node:* count 100` → collect keys
2. For each key: `GET` → unmarshal → compute status
3. Return all with status+total

### 3.4 GetNode (new)

1. `GET dagflow:node:<nodeId>` → unmarshal → compute status
2. Return node detail or "not found" error

### 3.5 GetNodesForFunc (refactored)

1. `SMEMBERS dagflow:func:<funcName>` → node IDs
2. For each ID: `GET dagflow:node:<id>` → filter by existence
3. Return matches (caller handles load balancing)

## 4. Frontend

### 4.1 Component tree

```
NodeRegistryPage
├── SearchBar (filter by nodeId or funcName)
├── StatsRow (total / online / offline counts)
├── RefreshControls (auto-refresh toggle, interval selector, manual refresh btn)
└── NodeTable
    ├── Node ID column
    ├── Address column
    ├── Functions column (Badge components)
    ├── Status column (green dot "Online" / red dot "Offline")
    └── Last Heartbeat column (relative time)
```

### 4.2 State management (zustand)

```typescript
interface NodeRegistryState {
  nodes: NodeDetail[];
  loading: boolean;
  search: string;
  autoRefresh: boolean;
  refreshInterval: number; // seconds
  fetchNodes: () => Promise<void>;
  setSearch: (s: string) => void;
  toggleAutoRefresh: () => void;
}
```

### 4.3 Routes

- `/nodes` → NodeRegistryPage
- Sidebar: "节点管理" entry with a server/monitor icon

## 5. Testing Strategy

- **Unit tests**: `registry_test.go` — unified storage CRUD, status derivation, function index
- **Integration**: existing `integration_test.go` in `remote_executor/` — ensure Register/Heartbeat backward compat
- **Frontend**: manual visual verification (table rendering, search, auto-refresh)

## 6. Backward Compatibility

- `NodeInfo` struct: add `LastHeartbeat` field (JSON deserialization tolerates missing field → 0)
- `GetNodesForFunc`: return format unchanged (still `[]NodeInfo`)
- `Register`/`Heartbeat` proto messages: unchanged
