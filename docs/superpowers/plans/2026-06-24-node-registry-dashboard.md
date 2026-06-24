---
change: node-registry-dashboard
design-doc: docs/superpowers/specs/2026-06-24-node-registry-dashboard-design.md
base-ref: 1e7dcefa4d78e59159c1a84f759bc6fa024e70dd
---

# Node Registry Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add API + frontend dashboard to visualize remote function nodes registered with DAGFlow, with unified Redis storage using `lastHeartbeat` for online/offline detection.

**Architecture:** Extend existing `NodeRegistry` gRPC service with `ListNodes`/`GetNode` RPCs. Refactor Redis storage to a single key per node with `lastHeartbeat` field plus function-name index SETs. Add HTTP routes via existing gRPC-handler bridge pattern. Build React table page with zustand store.

**Tech Stack:** Go 1.24, protobuf/gRPC, Redis (go-redis v9 via common-tools v2.RedisClient), React 19, TypeScript, zustand, Tailwind CSS, axios

## Global Constraints

- Use `v2.RedisClient` interface from common-tools for Redis operations (never `redis.Client` directly)
- All HTTP routes go through `router.go` using `engine.GRPC()` handler bridge
- Proto files in `backend/proto/remote_executor/`, generated Go in same package
- Frontend uses React Query-style manual fetch with zustand stores (existing pattern)
- Routes: `/nodes` for dashboard page
- Go test files in same package as implementation

---

### Task 1: Proto Definition

**Files:**
- Modify: `backend/proto/remote_executor/remote_executor.proto`
- Modify: `backend/proto/remote_executor/remote_executor_grpc.pb.go` (regenerated)
- Modify: `backend/proto/remote_executor/remote_executor.pb.go` (regenerated)

**Interfaces:**
- Produces: `ListNodesRequest`, `ListNodesResponse`, `GetNodeRequest`, `GetNodeResponse`, `NodeDetail` proto messages
- Produces: `NodeRegistryServer` interface extended with `ListNodes`/`GetNode`

- [ ] **Step 1: Add proto messages and RPCs**

```protobuf
// Add to remote_executor.proto, after the existing HeartbeatResponse message:

message NodeDetail {
  string node_id = 1;
  string address = 2;
  repeated string functions = 3;
  string status = 4;
  int64 last_heartbeat = 5;
}

message ListNodesRequest {}

message ListNodesResponse {
  repeated NodeDetail items = 1;
  int32 total = 2;
}

message GetNodeRequest {
  string node_id = 1;
}

message GetNodeResponse {
  NodeDetail node = 1;
}

// Add to service NodeRegistry:
service NodeRegistry {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc ListNodes(ListNodesRequest) returns (ListNodesResponse);
  rpc GetNode(GetNodeRequest) returns (GetNodeResponse);
}
```

- [ ] **Step 2: Regenerate Go code**

```bash
cd backend && protoc --go_out=. --go-grpc_out=. proto/remote_executor/remote_executor.proto
```

- [ ] **Step 3: Verify generated code compiles**

```bash
cd backend && go build ./proto/remote_executor/...
```

- [ ] **Step 4: Commit**

```bash
git add backend/proto/remote_executor/
git commit -m "feat: add ListNodes/GetNode RPCs to NodeRegistry proto"
```

---

### Task 2: Unified Node Storage Refactoring

**Files:**
- Modify: `backend/internal/node_registry/registry.go`

**Interfaces:**
- Consumes: `v2.RedisClient`, proto messages from Task 1
- Produces: updated `NodeInfo` with `LastHeartbeat`, refactored `Register`/`Heartbeat`, new `ListNodes`/`GetNode`

- [ ] **Step 1: Update constants and NodeInfo struct**

```go
package node_registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v2 "github.com/caiflower/common-tools/redis/v2"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
)

const (
	keyPrefixNode       = "dagflow:node:"
	keyPrefixFunc       = "dagflow:func:"
	nodeTTL             = 5 * time.Minute
	heartbeatThreshold  = 30 * time.Second
)

type NodeInfo struct {
	NodeID        string   `json:"nodeId"`
	Address       string   `json:"address"`
	Functions     []string `json:"functions"`
	LastHeartbeat int64    `json:"lastHeartbeat"`
}
```

- [ ] **Step 2: Refactor Register to use unified key + function index**

```go
func (r *NodeRegistry) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	now := time.Now().Unix()
	info := NodeInfo{
		NodeID:        req.NodeId,
		Address:       req.Address,
		Functions:     req.Functions,
		LastHeartbeat: now,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal node info: %w", err)
	}

	nodeKey := keyPrefixNode + req.NodeId
	if err := r.redis.Cmd().Set(ctx, nodeKey, data, nodeTTL).Err(); err != nil {
		return nil, fmt.Errorf("redis set node: %w", err)
	}

	for _, fn := range req.Functions {
		funcKey := keyPrefixFunc + fn
		if err := r.redis.Cmd().SAdd(ctx, funcKey, req.NodeId).Err(); err != nil {
			return nil, fmt.Errorf("redis sadd func index: %w", err)
		}
	}

	return &pb.RegisterResponse{Ok: true}, nil
}
```

- [ ] **Step 3: Refactor Heartbeat to update lastHeartbeat field**

```go
func (r *NodeRegistry) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	nodeKey := keyPrefixNode + req.NodeId

	data, err := r.redis.Cmd().Get(ctx, nodeKey).Bytes()
	if err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("node not found: %s", req.NodeId)
	}

	var info NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("unmarshal node info: %w", err)
	}

	info.LastHeartbeat = time.Now().Unix()

	newData, err := json.Marshal(info)
	if err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("marshal node info: %w", err)
	}

	if err := r.redis.Cmd().Set(ctx, nodeKey, newData, nodeTTL).Err(); err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("redis set heartbeat: %w", err)
	}

	return &pb.HeartbeatResponse{Ok: true}, nil
}
```

- [ ] **Step 4: Implement ListNodes**

```go
func (r *NodeRegistry) ListNodes(ctx context.Context, _ *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	keys, _, err := r.redis.Cmd().Scan(ctx, 0, keyPrefixNode+"*", 100).Result()
	if err != nil {
		return nil, fmt.Errorf("redis scan nodes: %w", err)
	}

	now := time.Now().Unix()
	var items []*pb.NodeDetail

	for _, key := range keys {
		data, err := r.redis.Cmd().Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var info NodeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		status := "offline"
		if now-info.LastHeartbeat < int64(heartbeatThreshold.Seconds()) {
			status = "online"
		}
		items = append(items, &pb.NodeDetail{
			NodeId:        info.NodeID,
			Address:       info.Address,
			Functions:     info.Functions,
			Status:        status,
			LastHeartbeat: info.LastHeartbeat,
		})
	}

	return &pb.ListNodesResponse{Items: items, Total: int32(len(items))}, nil
}
```

- [ ] **Step 5: Implement GetNode**

```go
func (r *NodeRegistry) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	nodeKey := keyPrefixNode + req.NodeId

	data, err := r.redis.Cmd().Get(ctx, nodeKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("node %q not found", req.NodeId)
	}

	var info NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal node info: %w", err)
	}

	now := time.Now().Unix()
	status := "offline"
	if now-info.LastHeartbeat < int64(heartbeatThreshold.Seconds()) {
		status = "online"
	}

	return &pb.GetNodeResponse{
		Node: &pb.NodeDetail{
			NodeId:        info.NodeID,
			Address:       info.Address,
			Functions:     info.Functions,
			Status:        status,
			LastHeartbeat: info.LastHeartbeat,
		},
	}, nil
}
```

- [ ] **Step 6: Refactor GetNodesForFunc to use function index**

```go
func (r *NodeRegistry) GetNodesForFunc(ctx context.Context, funcName string) ([]NodeInfo, error) {
	funcKey := keyPrefixFunc + funcName
	nodeIDs, err := r.redis.Cmd().SMembers(ctx, funcKey).Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers func index: %w", err)
	}

	now := time.Now().Unix()
	var nodes []NodeInfo
	for _, nodeID := range nodeIDs {
		nodeKey := keyPrefixNode + nodeID
		data, err := r.redis.Cmd().Get(ctx, nodeKey).Bytes()
		if err != nil {
			continue // node expired, skip
		}
		var info NodeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		if now-info.LastHeartbeat < int64(heartbeatThreshold.Seconds()) {
			nodes = append(nodes, info)
		}
	}
	return nodes, nil
}
```

- [ ] **Step 7: Add UnimplementedNodeRegistryServer methods (if not auto-generated)**

```go
// Add to registry.go if the generated _grpc.pb.go doesn't include stub methods:
func (r *NodeRegistry) mustEmbedUnimplementedNodeRegistryServer() {}
```

- [ ] **Step 8: Build to verify compilation**

```bash
cd backend && go build ./internal/node_registry/...
```

- [ ] **Step 9: Commit**

```bash
git add backend/internal/node_registry/registry.go
git commit -m "refactor: unified node storage with lastHeartbeat and function index"
```

---

### Task 3: Backend Tests

**Files:**
- Modify: `backend/internal/node_registry/registry_test.go`

**Interfaces:**
- Consumes: `NodeRegistry` methods from Task 2

- [ ] **Step 1: Update test helper for new NodeInfo**

```go
// In registry_test.go, update setupRegistry to handle new fields.
// The testRedisClient helper stays the same. Only tests change.
```

- [ ] **Step 2: Write TestRegisterStoresLastHeartbeat**

```go
func TestRegisterStoresLastHeartbeat(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	resp, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)
	assert.True(t, resp.Ok)

	// Verify via ListNodes that lastHeartbeat is set
	listResp, err := reg.ListNodes(ctx, &pb.ListNodesRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Items, 1)
	assert.Equal(t, "node-1", listResp.Items[0].NodeId)
	assert.Equal(t, "online", listResp.Items[0].Status)
	assert.Greater(t, listResp.Items[0].LastHeartbeat, int64(0))
}
```

- [ ] **Step 3: Write TestListNodesWithOfflineNode**

```go
func TestListNodesWithOfflineNode(t *testing.T) {
	reg, mr := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// Fast forward past heartbeat threshold but within node TTL
	mr.FastForward(31 * time.Second)

	listResp, err := reg.ListNodes(ctx, &pb.ListNodesRequest{})
	require.NoError(t, err)
	require.Len(t, listResp.Items, 1)
	assert.Equal(t, "offline", listResp.Items[0].Status)
}
```

- [ ] **Step 4: Write TestGetNode**

```go
func TestGetNode(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA", "funcB"},
	})
	require.NoError(t, err)

	resp, err := reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "node-1"})
	require.NoError(t, err)
	assert.Equal(t, "node-1", resp.Node.NodeId)
	assert.Equal(t, "localhost:50052", resp.Node.Address)
	assert.Equal(t, "online", resp.Node.Status)
	assert.Contains(t, resp.Node.Functions, "funcA")
	assert.Contains(t, resp.Node.Functions, "funcB")
}
```

- [ ] **Step 5: Write TestGetNodeNotFound**

```go
func TestGetNodeNotFound(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.GetNode(ctx, &pb.GetNodeRequest{NodeId: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
```

- [ ] **Step 6: Write TestFunctionIndex**

```go
func TestFunctionIndex(t *testing.T) {
	reg, _ := setupRegistry(t)
	ctx := context.Background()

	_, err := reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-1",
		Address:   "localhost:50052",
		Functions: []string{"funcA", "funcB"},
	})
	require.NoError(t, err)

	_, err = reg.Register(ctx, &pb.RegisterRequest{
		NodeId:    "node-2",
		Address:   "localhost:50053",
		Functions: []string{"funcA"},
	})
	require.NoError(t, err)

	// GetNodesForFunc should return both nodes for funcA
	nodes, err := reg.GetNodesForFunc(ctx, "funcA")
	require.NoError(t, err)
	assert.Len(t, nodes, 2)
}
```

- [ ] **Step 7: Update existing tests for backward compat**

Existing `TestRegisterStoresNodeInfo`, `TestHeartbeatRefreshesTTL`, `TestGetNodesForFuncNoMatch`, `TestNodeExpiresAfterTTL` should still pass with the refactored implementation. Update if needed.

- [ ] **Step 8: Run tests**

```bash
cd backend && go test ./internal/node_registry/... -v
```

Expected: all tests PASS

- [ ] **Step 9: Commit**

```bash
git add backend/internal/node_registry/registry_test.go
git commit -m "test: add tests for ListNodes, GetNode, function index, offline status"
```

---

### Task 4: HTTP Routes

**Files:**
- Modify: `backend/internal/api/router.go`

**Interfaces:**
- Consumes: NodeRegistry gRPC service from Task 2, proto handler functions from Task 1
- Produces: `GET /api/v1/nodes`, `GET /api/v1/nodes/:id`

- [ ] **Step 1: Add NodeRegistry gRPC handler functions**

In `backend/internal/proto/dagflow_grpc.pb.go` or create `backend/internal/proto/handlers.go`, add handler functions. Actually, since the NodeRegistry service is defined in `remote_executor.proto`, the generated handlers are in `proto/remote_executor/`. The existing pattern in `router.go` uses handler functions registered via `pb.XXX_Handler`. However, the NodeRegistry is not in `dagflow.proto`.

We need to create HTTP handler adapters in `router.go` that bridge to the NodeRegistry service. Since the existing `engine.GRPC()` pattern only works for services defined in `dagflow.proto`, we'll add simple HTTP handlers directly.

- [ ] **Step 1: Add routes to router.go**

```go
// In router.go, add after existing routes:

// ===== Node Registry Service =====
v1.GET("/nodes", func(ctx context.Context) (interface{}, error) {
    if nodeReg == nil {
        return nil, fmt.Errorf("node registry not initialized")
    }
    resp, err := nodeReg.ListNodes(ctx, &remote_executor.ListNodesRequest{})
    if err != nil {
        return nil, err
    }
    return resp, nil
})
v1.GET("/nodes/:id", func(ctx context.Context, id string) (interface{}, error) {
    if nodeReg == nil {
        return nil, fmt.Errorf("node registry not initialized")
    }
    resp, err := nodeReg.GetNode(ctx, &remote_executor.GetNodeRequest{NodeId: id})
    if err != nil {
        return nil, err
    }
    return resp.Node, nil
})
```

The `nodeReg` variable is a package-level var set from main.go similar to how `flowGrpcSvc` etc. are set.

- [ ] **Step 2: Add nodeReg variable and setter in services.go or a new file**

```go
// In router.go or a new file:
var nodeRegSvc *node_registry.NodeRegistry

func SetNodeRegistryService(svc *node_registry.NodeRegistry) {
    nodeRegSvc = svc
}
```

- [ ] **Step 3: Update main.go to call SetNodeRegistryService**

In `main.go` `initGrpcServices()`, after `nodeReg = node_registry.NewNodeRegistry(redisClient)`:
```go
api.SetNodeRegistryService(nodeReg)
```

- [ ] **Step 4: Build and verify compilation**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/router.go backend/cmd/server/main.go
git commit -m "feat: add HTTP routes for node registry list/get"
```

---

### Task 5: Frontend — API Client

**Files:**
- Modify: `frontend/src/api/client.ts`

**Interfaces:**
- Produces: `getNodes()`, `getNode(id)` functions

- [ ] **Step 1: Add NodeDetail type and API functions**

```typescript
// In client.ts, add:

export interface NodeDetail {
  nodeId: string;
  address: string;
  functions: string[];
  status: "online" | "offline";
  lastHeartbeat: number;
}

export interface ListNodesResponse {
  items: NodeDetail[];
  total: number;
}

export async function getNodes(): Promise<ListNodesResponse> {
  const res = await api.get<ListNodesResponse>("/nodes");
  return res.data;
}

export async function getNode(id: string): Promise<NodeDetail> {
  const res = await api.get<NodeDetail>(`/nodes/${id}`);
  return res.data;
}
```

Where `api` is the existing axios instance in `client.ts`.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/api/client.ts
git commit -m "feat: add node registry API client functions"
```

---

### Task 6: Frontend — Node Registry Page

**Files:**
- Create: `frontend/src/pages/NodeRegistryPage.tsx`
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/components/layout/Layout.tsx`

**Interfaces:**
- Consumes: `getNodes` from Task 5

- [ ] **Step 1: Create zustand store**

Create `frontend/src/store/nodeRegistryStore.ts`:

```typescript
import { create } from "zustand";
import { getNodes, NodeDetail } from "../api/client";

interface NodeRegistryState {
  nodes: NodeDetail[];
  loading: boolean;
  error: string | null;
  search: string;
  autoRefresh: boolean;
  refreshInterval: number;
  fetchNodes: () => Promise<void>;
  setSearch: (s: string) => void;
  setAutoRefresh: (v: boolean) => void;
  setRefreshInterval: (v: number) => void;
}

export const useNodeRegistryStore = create<NodeRegistryState>((set, get) => ({
  nodes: [],
  loading: false,
  error: null,
  search: "",
  autoRefresh: true,
  refreshInterval: 5,

  fetchNodes: async () => {
    set({ loading: true, error: null });
    try {
      const data = await getNodes();
      set({ nodes: data.items || [], loading: false });
    } catch (err: any) {
      set({ error: err.message || "Failed to fetch nodes", loading: false });
    }
  },

  setSearch: (search) => set({ search }),
  setAutoRefresh: (autoRefresh) => set({ autoRefresh }),
  setRefreshInterval: (refreshInterval) => set({ refreshInterval }),
}));
```

- [ ] **Step 2: Create NodeRegistryPage component**

Create `frontend/src/pages/NodeRegistryPage.tsx`:

```tsx
import { useEffect, useMemo, useRef } from "react";
import { useNodeRegistryStore } from "../store/nodeRegistryStore";
import { Badge } from "../components/ui/Badge";
import { Input } from "../components/ui/Input";
import { Button } from "../components/ui/Button";
import { RefreshCw, Server } from "lucide-react";

function formatHeartbeat(ts: number): string {
  const diff = Math.floor((Date.now() - ts * 1000) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  return `${Math.floor(diff / 3600)}h ago`;
}

export default function NodeRegistryPage() {
  const {
    nodes, loading, error, search, autoRefresh, refreshInterval,
    fetchNodes, setSearch, setAutoRefresh, setRefreshInterval,
  } = useNodeRegistryStore();

  const intervalRef = useRef<ReturnType<typeof setInterval>>();

  useEffect(() => {
    fetchNodes();
  }, []);

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(fetchNodes, refreshInterval * 1000);
    }
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [autoRefresh, refreshInterval]);

  const filtered = useMemo(() => {
    if (!search) return nodes;
    const s = search.toLowerCase();
    return nodes.filter(
      (n) =>
        n.nodeId.toLowerCase().includes(s) ||
        n.functions.some((f) => f.toLowerCase().includes(s))
    );
  }, [nodes, search]);

  const onlineCount = nodes.filter((n) => n.status === "online").length;
  const offlineCount = nodes.length - onlineCount;

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Node Registry</h1>
          <p className="text-gray-500 mt-1">Registered remote function nodes</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <label className="text-sm text-gray-600">Auto-refresh</label>
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="w-4 h-4"
            />
            <select
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
              className="border rounded px-2 py-1 text-sm"
            >
              <option value={5}>5s</option>
              <option value={10}>10s</option>
              <option value={30}>30s</option>
              <option value={60}>60s</option>
            </select>
          </div>
          <Button onClick={fetchNodes} disabled={loading} variant="outline" size="sm">
            <RefreshCw className={`w-4 h-4 mr-1 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="flex gap-6 text-sm">
        <span>Total: <strong>{nodes.length}</strong></span>
        <span className="text-green-600">Online: <strong>{onlineCount}</strong></span>
        <span className="text-red-500">Offline: <strong>{offlineCount}</strong></span>
      </div>

      {/* Search */}
      <Input
        placeholder="Search by node ID or function name..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="max-w-md"
      />

      {/* Error */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded">
          {error}
        </div>
      )}

      {/* Table */}
      <div className="border rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="text-left px-4 py-3 font-medium">Node ID</th>
              <th className="text-left px-4 py-3 font-medium">Address</th>
              <th className="text-left px-4 py-3 font-medium">Functions</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="text-left px-4 py-3 font-medium">Last Heartbeat</th>
            </tr>
          </thead>
          <tbody>
            {filtered.length === 0 && !loading && (
              <tr>
                <td colSpan={5} className="text-center py-12 text-gray-400">
                  <Server className="w-12 h-12 mx-auto mb-2 opacity-30" />
                  {search ? "No nodes match your search" : "No nodes registered yet"}
                </td>
              </tr>
            )}
            {filtered.map((node) => (
              <tr key={node.nodeId} className="border-b hover:bg-gray-50">
                <td className="px-4 py-3 font-mono text-xs">{node.nodeId}</td>
                <td className="px-4 py-3 font-mono text-xs">{node.address}</td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap gap-1">
                    {node.functions.map((fn) => (
                      <Badge key={fn} variant="secondary">{fn}</Badge>
                    ))}
                  </div>
                </td>
                <td className="px-4 py-3">
                  <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${
                    node.status === "online"
                      ? "bg-green-50 text-green-700"
                      : "bg-red-50 text-red-700"
                  }`}>
                    <span className={`w-1.5 h-1.5 rounded-full ${
                      node.status === "online" ? "bg-green-500" : "bg-red-500"
                    }`} />
                    {node.status === "online" ? "Online" : "Offline"}
                  </span>
                </td>
                <td className="px-4 py-3 text-gray-500">
                  {formatHeartbeat(node.lastHeartbeat)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Add route in App.tsx**

```tsx
// In App.tsx, add import and route:
import NodeRegistryPage from "./pages/NodeRegistryPage";

// In routes:
<Route path="/nodes" element={<NodeRegistryPage />} />
```

- [ ] **Step 4: Add sidebar entry in Layout.tsx**

```tsx
// In the sidebar navigation items array, add:
{
  to: "/nodes",
  icon: Server,       // from lucide-react
  label: "节点管理",
}
```

- [ ] **Step 5: Run frontend build to verify**

```bash
cd frontend && npm run build
```

Expected: build succeeds with no TypeScript errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/
git commit -m "feat: add Node Registry dashboard page with search and auto-refresh"
```

---

### Task 7: Final Verification

- [ ] **Step 1: Run all backend tests**

```bash
cd backend && go test ./internal/... ./taskx/... -count=1
```

Expected: all tests PASS

- [ ] **Step 2: Run backend build**

```bash
cd backend && go build ./cmd/server/...
```

Expected: build succeeds

- [ ] **Step 3: Run frontend build**

```bash
cd frontend && npm run build
```

Expected: build succeeds

- [ ] **Step 4: Mark tasks.md complete and commit**

```bash
# Update openspec/changes/node-registry-dashboard/tasks.md checkboxes
git add openspec/changes/node-registry-dashboard/tasks.md
git commit -m "chore: mark all tasks complete"
```
