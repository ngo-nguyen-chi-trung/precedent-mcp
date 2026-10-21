package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
)

type AddContextResult struct {
	Succeeded []int `json:"succeeded"`
	Failed    []int `json:"failed"`
}

func ContextToolCallEntryCountError(n int) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInvalidArgument, Message: fmt.Sprintf("entries must hold 1 to %d items, got %d", env.LimitContextEntries, n)}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}

type RetrieveContextResult struct {
	Results []RankedContext `json:"results"`
}

func (r *RetrieveContextResult) FromContextSearch(found []dbmodel.ContextSearch) {
	*r = RetrieveContextResult{Results: make([]RankedContext, 0, len(found))}
	for _, o := range found {
		r.Results = append(r.Results, RankedContext{ThreadID: o.ThreadID, Context: o.Content, Score: o.Metadata.Score})
	}
}

type RankedContext struct {
	ThreadID string  `json:"thread_id"`
	Context  string  `json:"context"`
	Score    float64 `json:"score"`
}

type GetContextThreadResult struct {
	ThreadID uuid.UUID       `json:"thread_id"`
	Entries  []ThreadContext `json:"entries"`
}

func (r *GetContextThreadResult) FromContextObjects(threadID uuid.UUID, found []dbmodel.ContextObject) {
	*r = GetContextThreadResult{ThreadID: threadID, Entries: make([]ThreadContext, 0, len(found))}
	for _, o := range found {
		r.Entries = append(r.Entries, ThreadContext{ThreadID: o.ThreadID, Context: o.Content, CreatedAt: o.CreatedAt})
	}
}

type ThreadContext struct {
	ThreadID  string    `json:"thread_id"`
	Context   string    `json:"context"`
	CreatedAt time.Time `json:"created_at"`
}

func ContextToolCallInternalErrorResponse(err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInternal, Message: err.Error()}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}
