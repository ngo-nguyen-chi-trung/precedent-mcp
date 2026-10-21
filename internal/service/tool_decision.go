package service

import (
	"cmp"
	"context"
	"encoding/json"
	"slices"

	"github.com/google/uuid"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/helper"
	"github.com/ngo/precedent_mcp/internal/service/model"
)

func (s *ToolService) RecordDecision(ctx context.Context, request model.RecordDecisionRequest) *pb.ToolCallResponse {
	obj, err := request.ToDecisionObject()
	if err != nil {
		return model.DecisionToolCallInvalidArgumentErrorResponse(err)
	}
	if err := s.decisions.Create(ctx, obj); err != nil {
		return model.DecisionToolCallFailedToCreateError(obj.ID, err)
	}
	if request.Supersedes != nil {
		superseded := dbmodel.DecisionObject{Status: string(env.StatusSuperseded), SupersededBy: &obj.ID}
		if err := s.decisions.Update(ctx, *request.Supersedes, superseded); err != nil {
			return model.DecisionToolCallFailedToSupersededError(obj.ID, *request.Supersedes, err)
		}
	}
	data, err := json.Marshal(model.RecordDecisionResult{DecisionID: obj.ID})
	if err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	return &pb.ToolCallResponse{Data: data}
}

func (s *ToolService) RetrieveDecisions(ctx context.Context, request model.RetrieveDecisionsRequest) *pb.ToolCallResponse {
	limit := int(env.LimitTopN)
	if request.TopN != nil {
		limit = min(*request.TopN, limit)
	}
	found, err := s.decisions.Search(ctx, request.Query, limit)
	if err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}

	var result model.RetrieveDecisionsResult
	if err := result.FromDecisionSearch(found); err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	result.Decisions = s.weightedReciprocalRankFusion(found, result.Decisions)

	data, err := json.Marshal(result)
	if err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	return &pb.ToolCallResponse{Data: data}
}

func (s *ToolService) ReviewDecision(ctx context.Context, request model.ReviewDecisionRequest) *pb.ToolCallResponse {
	o, err := s.decisions.Get(ctx, request.DecisionID)
	if err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	if o == nil {
		return model.DecisionToolCallNotFoundError(request.DecisionID)
	}
	obj, err := request.ToDecisionObject()
	if err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	if err := s.decisions.Update(ctx, request.DecisionID, obj); err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	data, err := json.Marshal(model.ReviewDecisionResult{DecisionID: request.DecisionID})
	if err != nil {
		return model.DecisionToolCallInternalErrorResponse(err)
	}
	return &pb.ToolCallResponse{Data: data}
}

func (s *ToolService) weightedReciprocalRankFusion(found []dbmodel.DecisionSearch, decisions []model.RankedDecision) []model.RankedDecision {
	byMatch := helper.RankedList[uuid.UUID]{Weight: env.WeightMatch}
	for _, o := range found {
		byMatch.Entries = append(byMatch.Entries, helper.RankEntry[uuid.UUID]{ID: o.Metadata.ID, Score: o.Metadata.Score})
	}
	byReview := helper.RankedList[uuid.UUID]{Weight: env.WeightReview}
	for _, d := range decisions {
		review := helper.RankEntry[uuid.UUID]{ID: d.DecisionID, Score: float64(env.RankNeutralReview)}
		if d.Review != nil {
			review.Score = float64(d.Review.Score)
		}
		byReview.Entries = append(byReview.Entries, review)
	}
	fused := helper.WeightedReciprocalRankFusion(byMatch, byReview)
	ranked := make([]model.RankedDecision, 0, len(decisions))
	for _, d := range decisions {
		d.Score = fused[d.DecisionID]
		ranked = append(ranked, d)
	}
	slices.SortStableFunc(ranked, func(a, b model.RankedDecision) int {
		return cmp.Compare(b.Score, a.Score)
	})
	return ranked
}
