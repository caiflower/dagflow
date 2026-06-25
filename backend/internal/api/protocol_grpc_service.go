package api

import (
	"context"

	pb "github.com/caiflower/dagflow/internal/proto"
	"github.com/caiflower/dagflow/internal/protocol"
)

// ProtocolGrpcService Protocol gRPC 服务实现
type ProtocolGrpcService struct {
	pb.UnimplementedProtocolServiceServer
	Registry *protocol.Registry `autowired:""`
}

func NewProtocolGrpcService() *ProtocolGrpcService {
	return &ProtocolGrpcService{}
}

func (s *ProtocolGrpcService) List(_ context.Context, _ *pb.ListProtocolRequest) (*pb.ListProtocolResponse, error) {
	infos := s.Registry.List()
	items := make([]*pb.ProtocolInfo, len(infos))
	for i, info := range infos {
		items[i] = protocolInfoToProto(info)
	}
	return &pb.ListProtocolResponse{Items: items}, nil
}

func (s *ProtocolGrpcService) Get(_ context.Context, req *pb.GetProtocolRequest) (*pb.ProtocolResponse, error) {
	factory, err := s.Registry.Get(req.Name)
	if err != nil {
		return nil, err
	}
	info := protocol.ProtocolInfo{
		Name:         factory.Name(),
		DisplayName:  factory.DisplayName(),
		Description:  factory.Description(),
		ConfigSchema: factory.ConfigSchema(),
	}
	return &pb.ProtocolResponse{Protocol: protocolInfoToProto(info)}, nil
}

func protocolInfoToProto(info protocol.ProtocolInfo) *pb.ProtocolInfo {
	fields := make([]*pb.ConfigField, len(info.ConfigSchema.Fields))
	for i, f := range info.ConfigSchema.Fields {
		fields[i] = &pb.ConfigField{
			Name:         f.Name,
			Label:        f.Label,
			Type:         f.Type,
			Required:     f.Required,
			DefaultValue: f.Default,
			Description:  f.Description,
			Options:      f.Options,
		}
	}
	return &pb.ProtocolInfo{
		Name:         info.Name,
		DisplayName:  info.DisplayName,
		Description:  info.Description,
		ConfigSchema: &pb.ConfigSchema{Fields: fields},
	}
}
