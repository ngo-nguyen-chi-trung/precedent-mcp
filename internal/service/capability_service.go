package service

import (
	"context"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
)

type CapabilityService struct {
	pb.UnimplementedCapabilityServiceServer
}

func NewCapabilityService() *CapabilityService {
	return &CapabilityService{}
}

func (s *CapabilityService) Discover(context.Context, *pb.Empty) (*pb.DiscoverResponse, error) {
	return &pb.DiscoverResponse{
		SupportedVersions: []string{string(env.VersionProtocol)},
		Extensions:        []string{string(env.ExtensionSkills)},
	}, nil
}
