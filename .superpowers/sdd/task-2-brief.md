### Task 2: Add Protocol/ProtocolConfig to SubtaskSettings

**Files:**
- Modify: `backend/taskx/dao/model/subtask.go` (or wherever `SubtaskSettings` is defined)

**Interfaces:**
- Produces: `SubtaskSettings.Protocol string`, `SubtaskSettings.ProtocolConfig map[string]any`

- [ ] **Step 1: Find SubtaskSettings definition**

Run: `grep -rn "type SubtaskSettings" backend/taskx/`

- [ ] **Step 2: Add fields**

```go
type SubtaskSettings struct {
	// ... existing fields ...
	Protocol       string         `json:"protocol,omitempty"`
	ProtocolConfig map[string]any `json:"protocolConfig,omitempty"`
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/dao/model/
git commit -m "feat(taskx): add Protocol and ProtocolConfig to SubtaskSettings"
```

---

