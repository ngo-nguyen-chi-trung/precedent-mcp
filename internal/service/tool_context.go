package service

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/google/uuid"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
)

func (s *ToolService) AddContext(ctx context.Context, request model.AddContextRequest) *pb.ToolCallResponse {
	if n := len(request.Entries); n == 0 || n > int(env.LimitContextEntries) {
		return model.ContextToolCallEntryCountError(n)
	}
	objects := request.ToContextObjects()
	batch := make([]dbmodel.ContextObject, 0, len(objects))
	positions := map[uuid.UUID][]int{}
	for i, o := range objects {
		if _, ok := positions[o.ID]; !ok {
			batch = append(batch, o)
		}
		positions[o.ID] = append(positions[o.ID], i)
	}
	succeeded, failed, err := s.entries.Create(ctx, batch)
	if err != nil {
		return model.ContextToolCallInternalErrorResponse(err)
	}
	data, err := json.Marshal(model.AddContextResult{
		Succeeded: entryPositions(succeeded, positions),
		Failed:    entryPositions(failed, positions),
	})
	if err != nil {
		return model.ContextToolCallInternalErrorResponse(err)
	}
	return &pb.ToolCallResponse{Data: data}
}

func entryPositions(ids []uuid.UUID, positions map[uuid.UUID][]int) []int {
	out := []int{}
	for _, id := range ids {
		out = append(out, positions[id]...)
	}
	slices.Sort(out)
	return out
}

func (s *ToolService) RetrieveContext(ctx context.Context, request model.RetrieveContextRequest) *pb.ToolCallResponse {
	limit := int(env.LimitTopN)
	if request.TopN != nil {
		limit = min(*request.TopN, limit)
	}
	found, err := s.entries.Search(ctx, request.Query, limit)
	if err != nil {
		return model.ContextToolCallInternalErrorResponse(err)
	}
	var result model.RetrieveContextResult
	result.FromContextSearch(found)
	data, err := json.Marshal(result)
	if err != nil {
		return model.ContextToolCallInternalErrorResponse(err)
	}
	return &pb.ToolCallResponse{Data: data}
}

func (s *ToolService) GetContextThread(ctx context.Context, request model.GetContextThreadRequest) *pb.ToolCallResponse {
	found, err := s.entries.Get(ctx, request.ThreadID.String(), int(env.LimitTopN))
	if err != nil {
		return model.ContextToolCallInternalErrorResponse(err)
	}
	var result model.GetContextThreadResult
	result.FromContextObjects(request.ThreadID, found)
	slices.SortStableFunc(result.Entries, func(a, b model.ThreadContext) int { return a.CreatedAt.Compare(b.CreatedAt) })
	data, err := json.Marshal(result)
	if err != nil {
		return model.ContextToolCallInternalErrorResponse(err)
	}
	return &pb.ToolCallResponse{Data: data}
}
