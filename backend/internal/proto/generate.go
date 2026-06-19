package proto

//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative dagflow.proto
//go:generate go run -mod=mod github.com/favadi/protoc-go-inject-tag -input dagflow.pb.go
