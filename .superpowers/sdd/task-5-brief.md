### Task 5: Clean up initByBean batch registration

**Files:**
- Modify: `backend/taskx/task.go` (initByBean method)

- [ ] **Step 1: Remove batch provider registration from initByBean**

Find and remove:
- `registerProvider` / `registerProviders` calls in `initByBean`
- Branch condition provider memory recovery (the `getRegisteredBranches` → `registerProvider` block)
- Any `registerRollbackProvider`, `registerPreProcessor`, `registerPostProcessor` in initByBean

- [ ] **Step 2: Remove related memory structures if only used in initByBean**

Check `getRegisteredBranches` — if only used in initByBean, remove it.

- [ ] **Step 3: Commit**

```bash
git add backend/taskx/task.go
git commit -m "refactor(taskx): remove batch provider registration from initByBean"
```

---

