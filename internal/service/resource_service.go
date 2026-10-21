package service

import (
	"context"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/asset"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/ngo/precedent_mcp/internal/service/validator"
)

type ResourceService struct {
	pb.UnimplementedResourceServiceServer
	skills   *asset.SkillStore
	examples *asset.ExampleStore
	pipe     validator.Pipeline[*model.ResourceURIRequest, *pb.ResourceReadResponse]
}

func NewResourceService(skills *asset.SkillStore, examples *asset.ExampleStore) *ResourceService {
	return &ResourceService{
		skills:   skills,
		examples: examples,
		pipe:     validator.NewBuilder[*model.ResourceURIRequest, *pb.ResourceReadResponse]().Add(validator.ResourceURI()).Build(),
	}
}

func (s *ResourceService) ListResources(context.Context, *pb.Empty) (*pb.ResourceListResponse, error) {
	res := &model.ResourceListResponse{}
	res.FromAssets(s.skills.ListMetadata(), s.examples.ListNames())
	return res.ResourceListResponse, nil
}

func (s *ResourceService) ReadResource(ctx context.Context, req *pb.ResourceReadRequest) (*pb.ResourceReadResponse, error) {
	r := &model.ResourceURIRequest{URI: req.GetUri()}
	if res, err := s.pipe.Run(ctx, r); res != nil || err != nil {
		return res, err
	}
	read := s.skills.Skill
	if r.Parsed.Scheme == string(env.SchemeExamples) {
		read = s.examples.Examples
	}
	text, err := read(r.Parsed.Host)
	if assetmodel.HasCode(err, env.AssetErrorSkillNotFound) || assetmodel.HasCode(err, env.AssetErrorExamplesNotFound) {
		return nil, model.ResourceNotFoundError(r.URI)
	}
	if err != nil {
		return nil, err
	}
	res := &model.ResourceReadResponse{}
	res.FromText(r.URI, text)
	return res.ResourceReadResponse, nil
}
