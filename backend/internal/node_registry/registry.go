package node_registry

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v2 "github.com/caiflower/common-tools/redis/v2"

	pb "github.com/caiflower/dagflow/proto/remote_executor"
)

const (
	keyPrefixNode      = "dagflow:node:"
	keyPrefixHeartbeat = "dagflow:node:heartbeat:"
	nodeTTL            = 30 * time.Second
)

type NodeInfo struct {
	NodeID    string   `json:"nodeId"`
	Address   string   `json:"address"`
	Functions []string `json:"functions"`
}

type NodeRegistry struct {
	pb.UnimplementedNodeRegistryServer
	redis v2.RedisClient
}

func NewNodeRegistry(client v2.RedisClient) *NodeRegistry {
	return &NodeRegistry{redis: client}
}

func (r *NodeRegistry) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	info := NodeInfo{
		NodeID:    req.NodeId,
		Address:   req.Address,
		Functions: req.Functions,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal node info: %w", err)
	}

	key := keyPrefixNode + req.NodeId
	if err := r.redis.Cmd().Set(ctx, key, data, nodeTTL).Err(); err != nil {
		return nil, fmt.Errorf("redis set node: %w", err)
	}

	hbKey := keyPrefixHeartbeat + req.NodeId
	if err := r.redis.Cmd().Set(ctx, hbKey, time.Now().Unix(), nodeTTL).Err(); err != nil {
		return nil, fmt.Errorf("redis set heartbeat: %w", err)
	}

	return &pb.RegisterResponse{Ok: true}, nil
}

func (r *NodeRegistry) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	hbKey := keyPrefixHeartbeat + req.NodeId
	if err := r.redis.Cmd().Set(ctx, hbKey, time.Now().Unix(), nodeTTL).Err(); err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("redis heartbeat: %w", err)
	}

	nodeKey := keyPrefixNode + req.NodeId
	r.redis.Cmd().Expire(ctx, nodeKey, nodeTTL)

	return &pb.HeartbeatResponse{Ok: true}, nil
}

func (r *NodeRegistry) GetNodesForFunc(ctx context.Context, funcName string) ([]NodeInfo, error) {
	keys, _, err := r.redis.Cmd().Scan(ctx, 0, keyPrefixNode+"*", 100).Result()
	if err != nil {
		return nil, fmt.Errorf("redis scan nodes: %w", err)
	}

	var nodes []NodeInfo
	for _, key := range keys {
		data, err := r.redis.Cmd().Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var info NodeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		for _, fn := range info.Functions {
			if fn == funcName {
				nodes = append(nodes, info)
				break
			}
		}
	}
	return nodes, nil
}
