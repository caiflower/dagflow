## Tasks

- [ ] Rename `NodeRegistryPort` → `Port` in `constants/config.go` and all references
- [ ] Update `etc/config.yaml` and `etc/default.yaml` config keys
- [ ] Create `GrpcServer` daemon struct implementing `global.DaemonResource`
- [ ] Register all 4 gRPC services on unified server
- [ ] Replace `startNodeRegistryServer()` with daemon-based startup in `main.go`
- [ ] Add Flow/Protocol/Execution gRPC client methods to `clients/go/sdk.go`
- [ ] Add SDK tests for new gRPC client methods
- [ ] Run full test suite (`go test ./internal/... ./clients/...`)
