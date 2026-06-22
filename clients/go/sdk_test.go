package sdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/caiflower/dagflow/proto/remote_executor"
)

func TestRegisterAndHandlerDispatch(t *testing.T) {
	s := New(Config{NodeID: "test"})

	type MyInput struct{ Value int }
	type MyOutput struct{ Result int }

	Register(s, "double", func(ctx context.Context, in MyInput) (MyOutput, error) {
		return MyOutput{Result: in.Value * 2}, nil
	})

	s.mu.RLock()
	handler, ok := s.handlers["double"]
	s.mu.RUnlock()
	require.True(t, ok, "handler should be registered")

	output, err := handler(context.Background(), []byte(`{"Value":21}`))
	require.NoError(t, err)
	assert.Equal(t, `{"Result":42}`, string(output))
}

func TestMissingFuncReturnsError(t *testing.T) {
	s := New(Config{NodeID: "test"})
	server := &ExecutorServer{Sdk: s}

	resp, err := server.Execute(context.Background(), &pb.ExecuteRequest{
		FuncName: "nonexistent",
		Input:    []byte(`{}`),
	})

	if err == nil {
		assert.Contains(t, resp.Error, "not registered")
	}
}

func TestMultipleFunctionsRegistered(t *testing.T) {
	s := New(Config{NodeID: "test"})

	type Empty struct{}
	Register(s, "funcA", func(ctx context.Context, in Empty) (Empty, error) { return Empty{}, nil })
	Register(s, "funcB", func(ctx context.Context, in Empty) (Empty, error) { return Empty{}, nil })
	Register(s, "funcC", func(ctx context.Context, in Empty) (Empty, error) { return Empty{}, nil })

	funcs := s.functionList()
	assert.Len(t, funcs, 3)
	assert.Contains(t, funcs, "funcA")
	assert.Contains(t, funcs, "funcB")
	assert.Contains(t, funcs, "funcC")
}

func TestNewClient(t *testing.T) {
	// Test client creation with a dummy address (won't connect, just tests construction)
	client, err := NewClient("localhost:50051")
	if err == nil {
		defer client.Close()
	}
	// Connection may fail, but client struct should be created
	assert.NotNil(t, client)
	assert.NotNil(t, client.Flow)
	assert.NotNil(t, client.Protocol)
	assert.NotNil(t, client.Execution)
}
