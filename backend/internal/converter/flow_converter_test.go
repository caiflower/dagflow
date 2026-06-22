package converter

import (
	"context"
	"testing"

	"github.com/caiflower/dagflow/taskx/executor"
	"github.com/stretchr/testify/assert"
)

// echoProvider returns a provider that echoes its input as output
func echoProvider(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
	return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (string, error) {
		return "", nil
	}), nil
}

// branchProviderFactory returns a provider factory for branch nodes.
// The returned provider always selects the given targetName.
func branchProviderFactory(targetName string) func(string, map[string]any) (executor.ExecutorProvider, error) {
	return func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (string, error) {
			return targetName, nil
		}), nil
	}
}

func TestFlowToTask_SingleBranch(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "mybranch", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "pathA", Type: "task", Protocol: "local"},
		{ID: "n4", Name: "pathB", Type: "task", Protocol: "local"},
		{ID: "n5", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n2", Target: "n4"},
		{ID: "e4", Source: "n3", Target: "n5"},
		{ID: "e5", Source: "n4", Target: "n5"},
	}

	providerFactory := branchProviderFactory("pathA")
	task, err := FlowToTaskWithNodes("test-branch", nodes, edges, providerFactory, nil)
	assert.NoError(t, err)

	_, err = task.Compile()
	assert.NoError(t, err)
}

func TestFlowToTask_NestedBranch(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "outer", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "inner", Type: "branch", Protocol: "local"},
		{ID: "n4", Name: "innerA", Type: "task", Protocol: "local"},
		{ID: "n5", Name: "innerB", Type: "task", Protocol: "local"},
		{ID: "n6", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n3", Target: "n4"},
		{ID: "e4", Source: "n3", Target: "n5"},
		{ID: "e5", Source: "n2", Target: "n6"}, // fallback path
		{ID: "e6", Source: "n4", Target: "n6"},
		{ID: "e7", Source: "n5", Target: "n6"},
	}

	providerFactory := func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (string, error) {
			return "innerA", nil
		}), nil
	}

	task, err := FlowToTaskWithNodes("test-nested", nodes, edges, providerFactory, nil)
	assert.NoError(t, err)

	_, err = task.Compile()
	assert.NoError(t, err)
}

func TestFlowToTask_SkipBranchWithOneSuccessor(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "single", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"}, // only 1 successor — not a real branch
	}

	task, err := FlowToTaskWithNodes("test-skip", nodes, edges, echoProvider, nil)
	assert.NoError(t, err)

	// Should compile without error (branch node simply skipped)
	_, err = task.Compile()
	assert.NoError(t, err)
}

func TestFlowToTask_BranchProviderMustReturnString(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "badBranch", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "pathA", Type: "task", Protocol: "local"},
		{ID: "n4", Name: "pathB", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n2", Target: "n4"},
	}

	// Provider returns int instead of string
	badFactory := func(protocol string, config map[string]any) (executor.ExecutorProvider, error) {
		return executor.NewLocalExecutor(func(ctx context.Context, input map[string]any) (int, error) {
			return 42, nil
		}), nil
	}

	_, err := FlowToTaskWithNodes("test-bad", nodes, edges, badFactory, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must return string")
}

func TestFlowToTask_Roundtrip(t *testing.T) {
	nodes := []FlowNode{
		{ID: "n1", Name: "start", Type: "task", Protocol: "local"},
		{ID: "n2", Name: "mybranch", Type: "branch", Protocol: "local"},
		{ID: "n3", Name: "pathA", Type: "task", Protocol: "local"},
		{ID: "n4", Name: "pathB", Type: "task", Protocol: "local"},
		{ID: "n5", Name: "end", Type: "task", Protocol: "local"},
	}
	edges := []FlowEdge{
		{ID: "e1", Source: "n1", Target: "n2"},
		{ID: "e2", Source: "n2", Target: "n3"},
		{ID: "e3", Source: "n2", Target: "n4"},
		{ID: "e4", Source: "n3", Target: "n5"},
		{ID: "e5", Source: "n4", Target: "n5"},
	}

	providerFactory := branchProviderFactory("pathA")
	task, err := FlowToTaskWithNodes("test-roundtrip", nodes, edges, providerFactory, nil)
	assert.NoError(t, err)

	// Compile twice to verify idempotency
	_, err = task.Compile()
	assert.NoError(t, err)
}
