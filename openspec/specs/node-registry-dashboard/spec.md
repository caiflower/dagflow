## ADDED Requirements

### Requirement: Display registered nodes in a table
The frontend SHALL display all registered remote function nodes in a data-center-style table showing node ID, address, supported functions, and online/offline status.

#### Scenario: Table displays all nodes with mixed status
- **WHEN** the user navigates to the Node Registry page and there are 3 nodes registered
- **THEN** the page displays a table with 3 rows, each showing node ID, address, functions (comma-separated or badge list), and a green "Online" or red "Offline" status indicator

#### Scenario: Empty state
- **WHEN** the user navigates to the Node Registry page and no nodes are registered
- **THEN** the page displays an empty state message indicating no nodes are registered

### Requirement: Auto-refresh node list
The frontend SHALL support automatic refreshing of the node list at a configurable interval.

#### Scenario: Auto-refresh updates data
- **WHEN** auto-refresh is enabled with a 5-second interval
- **THEN** the node list is re-fetched every 5 seconds and the table updates automatically

#### Scenario: Manual refresh
- **WHEN** the user clicks the refresh button
- **THEN** the node list is immediately re-fetched and the table updates

### Requirement: Search and filter nodes
The frontend SHALL allow users to search nodes by node ID or function name.

#### Scenario: Filter by node ID
- **WHEN** the user types a partial node ID in the search box
- **THEN** the table filters to show only nodes whose ID contains the search text

#### Scenario: Filter by function name
- **WHEN** the user types a function name in the search box
- **THEN** the table filters to show only nodes that support the searched function

### Requirement: Navigation entry for node registry
The frontend SHALL include a navigation link to the Node Registry page in the sidebar.

#### Scenario: Sidebar navigation
- **WHEN** the user views the sidebar navigation
- **THEN** a "Node Registry" (节点管理) entry is visible and clicking it navigates to `/nodes`
