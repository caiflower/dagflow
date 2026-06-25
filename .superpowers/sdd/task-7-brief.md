### Task 7: Run existing tests, fix regressions

**Files:**
- All modified files

- [ ] **Step 1: Run full test suite**

```bash
cd backend && go test ./... --count=1
```

- [ ] **Step 2: Fix any failing tests**

Expected issues:
- Tests that relied on initByBean registering providers may need adjustment
- Tests that directly manipulated `_providerRegistry` may need updates

- [ ] **Step 3: Commit any test fixes**

```bash
git add -A
git commit -m "test: fix test regressions from distributed provider registry changes"
```

---

