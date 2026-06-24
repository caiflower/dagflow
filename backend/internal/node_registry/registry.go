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
	keyPrefixFunc      = "dagflow:func:"
	nodeTTL            = 5 * time.Minute
	heartbeatThreshold = 30 * time.Second
)

type NodeInfo struct {
	NodeID        string   `json:"nodeId"`
	Address       string   `json:"address"`
	Functions     []string `json:"functions"`
	LastHeartbeat int64    `json:"lastHeartbeat"`
}

type NodeRegistry struct {
	pb.UnimplementedNodeRegistryServer
	redis    v2.RedisClient
	timeFunc func() time.Time
}

func NewNodeRegistry(client v2.RedisClient) *NodeRegistry {
	return &NodeRegistry{redis: client, timeFunc: time.Now}
}

func (r *NodeRegistry) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	now := r.timeFunc().Unix()
	info := NodeInfo{
		NodeID:        req.NodeId,
		Address:       req.Address,
		Functions:     req.Functions,
		LastHeartbeat: now,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("marshal node info: %w", err)
	}

	nodeKey := keyPrefixNode + req.NodeId
	if err := r.redis.Cmd().Set(ctx, nodeKey, data, nodeTTL).Err(); err != nil {
		return nil, fmt.Errorf("redis set node: %w", err)
	}

	for _, fn := range req.Functions {
		funcKey := keyPrefixFunc + fn
		if err := r.redis.Cmd().SAdd(ctx, funcKey, req.NodeId).Err(); err != nil {
			return nil, fmt.Errorf("redis sadd func index: %w", err)
		}
	}

	return &pb.RegisterResponse{Ok: true}, nil
}

func (r *NodeRegistry) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	nodeKey := keyPrefixNode + req.NodeId

	data, err := r.redis.Cmd().Get(ctx, nodeKey).Bytes()
	if err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("node not found: %s", req.NodeId)
	}

	var info NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("unmarshal node info: %w", err)
	}

	info.LastHeartbeat = r.timeFunc().Unix()

	newData, err := json.Marshal(info)
	if err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("marshal node info: %w", err)
	}

	if err := r.redis.Cmd().Set(ctx, nodeKey, newData, nodeTTL).Err(); err != nil {
		return &pb.HeartbeatResponse{Ok: false}, fmt.Errorf("redis set heartbeat: %w", err)
	}

	return &pb.HeartbeatResponse{Ok: true}, nil
}

func (r *NodeRegistry) ListNodes(ctx context.Context, _ *pb.ListNodesRequest) (*pb.ListNodesResponse, error) {
	keys, _, err := r.redis.Cmd().Scan(ctx, 0, keyPrefixNode+"*", 100).Result()
	if err != nil {
		return nil, fmt.Errorf("redis scan nodes: %w", err)
	}

	now := r.timeFunc().Unix()
	var items []*pb.NodeDetail

	for _, key := range keys {
		data, err := r.redis.Cmd().Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var info NodeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		status := "offline"
		if now-info.LastHeartbeat < int64(heartbeatThreshold.Seconds()) {
			status = "online"
		}
		items = append(items, &pb.NodeDetail{
			NodeId:        info.NodeID,
			Address:       info.Address,
			Functions:     info.Functions,
			Status:        status,
			LastHeartbeat: info.LastHeartbeat,
		})
	}

	return &pb.ListNodesResponse{Items: items, Total: int32(len(items))}, nil
}

func (r *NodeRegistry) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	nodeKey := keyPrefixNode + req.NodeId

	data, err := r.redis.Cmd().Get(ctx, nodeKey).Bytes()
	if err != nil {
		return nil, fmt.Errorf("node %q not found", req.NodeId)
	}

	var info NodeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal node info: %w", err)
	}

	now := r.timeFunc().Unix()
	status := "offline"
	if now-info.LastHeartbeat < int64(heartbeatThreshold.Seconds()) {
		status = "online"
	}

	return &pb.GetNodeResponse{
		Node: &pb.NodeDetail{
			NodeId:        info.NodeID,
			Address:       info.Address,
			Functions:     info.Functions,
			Status:        status,
			LastHeartbeat: info.LastHeartbeat,
		},
	}, nil
}

func (r *NodeRegistry) GetNodesForFunc(ctx context.Context, funcName string) ([]NodeInfo, error) {
	funcKey := keyPrefixFunc + funcName
	nodeIDs, err := r.redis.Cmd().SMembers(ctx, funcKey).Result()
	if err != nil {
		return nil, fmt.Errorf("redis smembers func index: %w", err)
	}

	now := r.timeFunc().Unix()
	var nodes []NodeInfo
	for _, nodeID := range nodeIDs {
		nodeKey := keyPrefixNode + nodeID
		data, err := r.redis.Cmd().Get(ctx, nodeKey).Bytes()
		if err != nil {
			continue
		}
		var info NodeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		if now-info.LastHeartbeat < int64(heartbeatThreshold.Seconds()) {
			nodes = append(nodes, info)
		}
	}
	return nodes, nil
}
