## Tasks

- [x] Rename `NodeRegistryPort` → `Port` in `constants/config.go` and all references
- [x] Update `etc/config.yaml` and `etc/default.yaml` config keys
- [x] Create `GrpcServer` daemon struct implementing `global.DaemonResource`
- [x] Register all 4 gRPC services on unified server
- [x] Replace `startNodeRegistryServer()` with daemon-based startup in `main.go`
- [x] Add Flow/Protocol/Execution gRPC client methods to `clients/go/sdk.go`
- [x] Add SDK tests for new gRPC client methods
- [x] Run full test suite (`go test ./internal/... ./clients/...`)
