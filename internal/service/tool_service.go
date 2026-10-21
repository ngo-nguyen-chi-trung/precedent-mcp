package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/asset"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/db"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/ngo/precedent_mcp/internal/service/validator"
)

type ToolFunc func(ctx context.Context, args []byte) *pb.ToolCallResponse

type ToolService struct {
	pb.UnimplementedToolServiceServer
	tools     map[env.ToolName]ToolFunc
	pipe      validator.Pipeline[*model.ToolCallRequest, *pb.ToolCallResponse]
	store     *asset.ToolStore
	entries   db.ContextStore
	decisions db.DecisionStore
}

func NewToolService(store *asset.ToolStore, entries db.ContextStore, decisions db.DecisionStore) *ToolService {
	tools := map[env.ToolName]ToolFunc{}
	return &ToolService{
		tools: tools,
		pipe: validator.NewBuilder[*model.ToolCallRequest, *pb.ToolCallResponse]().
			Add(validator.KnownTool(tools, store)).
			Add(validator.Arguments()).
			Build(),
		store:     store,
		entries:   entries,
		decisions: decisions,
	}
}

func (s *ToolService) ListTools(context.Context, *pb.Empty) (*pb.ToolListResponse, error) {
	entries, err := s.store.Tools()
	if err != nil {
		return nil, err
	}
	res := &model.ToolListResponse{}
	if err := res.FromToolEntries(entries); err != nil {
		return nil, err
	}
	return res.ToolListResponse, nil
}

func (s *ToolService) CallTool(ctx context.Context, req *pb.ToolCallRequest) (*pb.ToolCallResponse, error) {
	r := &model.ToolCallRequest{ToolCallRequest: req}
	if res, err := s.pipe.Run(ctx, r); res != nil || err != nil {
		return res, err
	}
	return s.tools[env.ToolName(r.GetName())](ctx, r.GetArguments()), nil
}

func (s *ToolService) Register(tools map[env.ToolName]ToolFunc) error {
	entries, err := s.store.Tools()
	if err != nil {
		return err
	}
	for name, fn := range tools {
		if !slices.ContainsFunc(entries, func(entry assetmodel.ToolEntry) bool { return entry.Name == string(name) }) {
			return fmt.Errorf("tool %s has no definition in %s", name, env.PathToolList)
		}
		s.tools[name] = fn
	}
	return nil
}

func Tool[A any](fn func(context.Context, A) *pb.ToolCallResponse) ToolFunc {
	return func(ctx context.Context, args []byte) *pb.ToolCallResponse {
		var request A
		if err := json.Unmarshal(args, &request); err != nil {
			return model.InvalidArgsJSONError(err)
		}
		return fn(ctx, request)
	}
}
