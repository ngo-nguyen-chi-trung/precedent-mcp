package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"go.yaml.in/yaml/v3"
)

type RecordDecisionResult struct {
	DecisionID uuid.UUID `json:"decision_id"`
}

func DecisionToolCallInvalidArgumentErrorResponse(err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInvalidArgument, Message: err.Error()}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}

func DecisionToolCallFailedToCreateError(id uuid.UUID, err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInternal, Message: fmt.Sprintf("create decision %s: %v", id, err)}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}

func DecisionToolCallFailedToSupersededError(created, superseded uuid.UUID, err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInternal, Message: fmt.Sprintf("decision %s created, but supersede decision %s failed: %v", created, superseded, err)}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}

type RetrieveDecisionsResult struct {
	Decisions []RankedDecision `json:"decisions"`
}

func (r *RetrieveDecisionsResult) FromDecisionSearch(found []dbmodel.DecisionSearch) error {
	*r = RetrieveDecisionsResult{Decisions: make([]RankedDecision, 0, len(found))}
	for _, o := range found {
		d := RankedDecision{
			DecisionID:   o.Metadata.ID,
			Title:        o.Title,
			Status:       env.DecisionStatus(o.Status),
			Tags:         o.Tags,
			Supersedes:   o.Supersedes,
			SupersededBy: o.SupersededBy,
			CreatedAt:    o.CreatedAt,
		}
		if err := yaml.Unmarshal([]byte(o.Content), &d.Details); err != nil {
			return err
		}
		if err := yaml.Unmarshal([]byte(o.Review), &d.Review); err != nil {
			return err
		}
		r.Decisions = append(r.Decisions, d)
	}
	return nil
}

type RankedDecision struct {
	DecisionID   uuid.UUID          `json:"decision_id"`
	Title        string             `json:"title"`
	Status       env.DecisionStatus `json:"status"`
	Tags         []string           `json:"tags,omitempty"`
	Supersedes   *uuid.UUID         `json:"supersedes,omitempty"`
	SupersededBy *uuid.UUID         `json:"superseded_by,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	Details      DecisionDetails
	Review       *DecisionReview `json:"review,omitempty"`
	Score        float64         `json:"score"`
}

type ReviewDecisionResult struct {
	DecisionID uuid.UUID `json:"decision_id"`
}

func DecisionToolCallNotFoundError(id uuid.UUID) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorDecisionNotFound, Message: fmt.Sprintf("decision not found: %s", id)}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}

func DecisionToolCallInternalErrorResponse(err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInternal, Message: err.Error()}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}
