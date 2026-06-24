## ADDED Requirements

### Requirement: List all registered nodes
The system SHALL provide an API to list all registered remote function nodes with their registration info and health status derived from the `lastHeartbeat` field.

#### Scenario: List nodes with mixed status
- **WHEN** there are 3 nodes registered: 2 with `lastHeartbeat` within 30 seconds and 1 with `lastHeartbeat` older than 30 seconds
- **THEN** the API returns 3 nodes, with 2 marked "online" and 1 marked "offline"

#### Scenario: No nodes registered
- **WHEN** no nodes have registered
- **THEN** the API returns an empty list with total count 0

#### Scenario: List nodes via gRPC
- **WHEN** a gRPC client calls `ListNodes` with an empty request
- **THEN** the response contains all registered nodes with their ID, address, functions, status, and lastHeartbeat timestamp

#### Scenario: List nodes via HTTP
- **WHEN** an HTTP client sends `GET /api/v1/nodes`
- **THEN** the response contains all registered nodes in JSON format with ID, address, functions, status, and lastHeartbeat

### Requirement: Get single node detail
The system SHALL provide an API to retrieve detailed information about a specific registered node by its node ID.

#### Scenario: Get existing node
- **WHEN** requesting a node with an existing node ID `node-abc123`
- **THEN** the API returns the node's ID, address, functions list, status, and lastHeartbeat timestamp

#### Scenario: Get non-existent node
- **WHEN** requesting a node with a non-existent node ID
- **THEN** the API returns a "not found" error

#### Scenario: Get node via gRPC
- **WHEN** a gRPC client calls `GetNode` with a valid node ID
- **THEN** the response contains the node's full information

#### Scenario: Get node via HTTP
- **WHEN** an HTTP client sends `GET /api/v1/nodes/:id` with a valid node ID
- **THEN** the response contains the node's full information in JSON format

### Requirement: Unified node storage with lastHeartbeat
The system SHALL store node registration info and heartbeat in a single Redis key per node, using a `lastHeartbeat` field to determine online/offline status.

#### Scenario: Registration creates unified key
- **WHEN** a node registers with `Register(nodeId, address, functions)`
- **THEN** a single Redis key `dagflow:node:<nodeId>` is created with `lastHeartbeat` set to current time and TTL of 5 minutes

#### Scenario: Heartbeat updates lastHeartbeat
- **WHEN** a node sends a heartbeat
- **THEN** the `lastHeartbeat` field is updated to current time and the key TTL is refreshed to 5 minutes

#### Scenario: Node is online
- **WHEN** the node key exists and `now - lastHeartbeat < 30s`
- **THEN** the node's status is reported as "online"

#### Scenario: Node is offline
- **WHEN** the node key exists and `now - lastHeartbeat >= 30s`
- **THEN** the node's status is reported as "offline" (heartbeat stopped but registration data persists)

#### Scenario: Node is gone
- **WHEN** the node key has expired (no heartbeat for > 5 minutes)
- **THEN** the node no longer appears in the list
