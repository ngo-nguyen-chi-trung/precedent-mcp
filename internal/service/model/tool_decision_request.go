package model

import (
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"go.yaml.in/yaml/v3"
)

type RecordDecisionRequest struct {
	Title      string             `json:"title"`
	Status     env.DecisionStatus `json:"status"`
	Tags       []string           `json:"tags"`
	Supersedes *uuid.UUID         `json:"supersedes"`
	Details    DecisionDetails
}

func (d RecordDecisionRequest) ToDecisionObject() (dbmodel.DecisionObject, error) {
	content, err := yaml.Marshal(d.Details)
	if err != nil {
		return dbmodel.DecisionObject{}, err
	}
	return dbmodel.DecisionObject{
		ID:         uuid.New(),
		Title:      d.Title,
		Status:     string(d.Status),
		Tags:       d.Tags,
		Supersedes: d.Supersedes,
		Content:    string(content),
		CreatedAt:  time.Now().UTC(),
	}, nil
}

type RetrieveDecisionsRequest struct {
	Query string `json:"query"`
	TopN  *int   `json:"top_n"`
}

type ReviewDecisionRequest struct {
	DecisionID uuid.UUID `json:"decision_id"`
	Score      int       `json:"score"`
	Reason     string    `json:"reason"`
}

func (r ReviewDecisionRequest) ToDecisionObject() (dbmodel.DecisionObject, error) {
	content, err := yaml.Marshal(DecisionReview{Score: r.Score, Reason: r.Reason, ReviewedAt: time.Now().UTC()})
	if err != nil {
		return dbmodel.DecisionObject{}, err
	}
	return dbmodel.DecisionObject{Review: string(content)}, nil
}

type DecisionDetails struct {
	Situation string           `json:"situation"           yaml:"situation"`
	Decision  string           `json:"decision"            yaml:"decision"`
	Rationale string           `json:"rationale"           yaml:"rationale"`
	Options   []DecisionOption `json:"options,omitempty"   yaml:"options,omitempty"`
	Tradeoffs []string         `json:"tradeoffs,omitempty" yaml:"tradeoffs,omitempty"`
	Facts     []SourcedNote    `json:"facts,omitempty"     yaml:"facts,omitempty"`
	Practices []SourcedNote    `json:"practices,omitempty" yaml:"practices,omitempty"`
}

type DecisionOption struct {
	Name    string   `json:"name"              yaml:"name"`
	Pros    []string `json:"pros,omitempty"    yaml:"pros,omitempty"`
	Cons    []string `json:"cons,omitempty"    yaml:"cons,omitempty"`
	Verdict string   `json:"verdict,omitempty" yaml:"verdict,omitempty"`
}

type SourcedNote struct {
	Text   string `json:"text"             yaml:"text"`
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

type DecisionReview struct {
	Score      int       `json:"score"       yaml:"score"`
	Reason     string    `json:"reason"      yaml:"reason"`
	ReviewedAt time.Time `json:"reviewed_at" yaml:"reviewed_at"`
}
