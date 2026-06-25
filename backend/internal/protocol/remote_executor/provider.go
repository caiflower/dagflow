package remote_executor

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/caiflower/dagflow/internal/node_registry"
	pb "github.com/caiflower/dagflow/proto/remote_executor"
	"github.com/caiflower/dagflow/taskx/executor"
)

const ProtocolRemoteFunc executor.Protocol = "remoteFunc"

type RemoteFuncProvider struct {
	FuncName string
	Timeout  time.Duration
	Registry *node_registry.NodeRegistry
	Pool     *ConnPool
}

func (p *RemoteFuncProvider) Execute(ctx context.Context, data *executor.TaskData) (any, error) {
	nodes, err := p.Registry.GetNodesForFunc(ctx, p.FuncName)
	if err != nil {
		return nil, fmt.Errorf("get nodes for func %q: %w", p.FuncName, err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no available nodes for function %q", p.FuncName)
	}

	node := nodes[rand.Intn(len(nodes))]

	conn, err := p.Pool.GetConn(node.Address)
	if err != nil {
		return nil, fmt.Errorf("get connection to %s: %w", node.Address, err)
	}
	defer p.Pool.Release(node.Address)

	client := pb.NewRemoteExecutorClient(conn)

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	input := []byte(data.Input)
	resp, err := client.Execute(ctx, &pb.ExecuteRequest{
		FuncName: p.FuncName,
		Input:    input,
		TaskId:   data.TaskId,
	})
	if err != nil {
		return nil, fmt.Errorf("remote execute %q: %w", p.FuncName, err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("remote function %q error: %s", p.FuncName, resp.Error)
	}

	return resp.Output, nil
}

func (p *RemoteFuncProvider) Protocol() executor.Protocol {
	return ProtocolRemoteFunc
}

func (p *RemoteFuncProvider) ProviderConfig() map[string]any {
	return map[string]any{
		"funcName": p.FuncName,
		"timeout":  int(p.Timeout.Seconds()),
	}
}
