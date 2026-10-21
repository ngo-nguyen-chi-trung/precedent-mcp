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

type SkillService struct {
	pb.UnimplementedSkillServiceServer
	store *asset.SkillStore
	pipe  validator.Pipeline[*model.ResourceURIRequest, *pb.SkillGetResponse]
}

func NewSkillService(store *asset.SkillStore) *SkillService {
	return &SkillService{
		store: store,
		pipe:  validator.NewBuilder[*model.ResourceURIRequest, *pb.SkillGetResponse]().Add(validator.SkillURI()).Build(),
	}
}

func (s *SkillService) ListSkills(context.Context, *pb.Empty) (*pb.SkillListResponse, error) {
	res := &model.SkillListResponse{}
	res.FromMetadata(s.store.ListMetadata())
	return res.SkillListResponse, nil
}

func (s *SkillService) GetSkill(ctx context.Context, req *pb.SkillGetRequest) (*pb.SkillGetResponse, error) {
	r := &model.ResourceURIRequest{URI: req.GetUri()}
	if res, err := s.pipe.Run(ctx, r); res != nil || err != nil {
		return res, err
	}
	md, err := s.store.Metadata(r.Parsed.Host)
	if assetmodel.HasCode(err, env.AssetErrorSkillNotFound) {
		return nil, model.ResourceNotFoundError(r.URI)
	}
	if err != nil {
		return nil, err
	}
	res := &model.SkillGetResponse{}
	res.FromMetadata(md)
	return res.SkillGetResponse, nil
}
