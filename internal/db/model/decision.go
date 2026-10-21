package model

import (
	"time"

	"github.com/google/uuid"
)

type DecisionObject struct {
	ID           uuid.UUID  `json:"-"                      weaviate:"-"      tokenization:"-"`
	Title        string     `json:"title,omitempty"        weaviate:"text"   tokenization:"word"`
	Status       string     `json:"status,omitempty"       weaviate:"text"   tokenization:"field"`
	Tags         []string   `json:"tags,omitempty"         weaviate:"text[]" tokenization:"field"`
	Supersedes   *uuid.UUID `json:"supersedes,omitempty"   weaviate:"text"   tokenization:"field"`
	SupersededBy *uuid.UUID `json:"supersededBy,omitempty" weaviate:"text"   tokenization:"field"`
	Content      string     `json:"content,omitempty"      weaviate:"text"   tokenization:"word"`
	Review       string     `json:"review,omitempty"       weaviate:"text"   tokenization:"word"`
	CreatedAt    time.Time  `json:"createdAt,omitzero"     weaviate:"date"`
}

type DecisionSearch struct {
	Title        string           `json:"title"                  weaviate:"text"   tokenization:"word"`
	Status       string           `json:"status"                 weaviate:"text"   tokenization:"field"`
	Tags         []string         `json:"tags,omitempty"         weaviate:"text[]" tokenization:"field"`
	Supersedes   *uuid.UUID       `json:"supersedes,omitempty"   weaviate:"text"   tokenization:"field"`
	SupersededBy *uuid.UUID       `json:"supersededBy,omitempty" weaviate:"text"   tokenization:"field"`
	Content      string           `json:"content"                weaviate:"text"   tokenization:"word"`
	Review       string           `json:"review,omitempty"       weaviate:"text"   tokenization:"word"`
	CreatedAt    time.Time        `json:"createdAt"              weaviate:"date"`
	Metadata     DecisionMetadata `json:"_additional"            nested:""`
}

type DecisionMetadata struct {
	ID    uuid.UUID `json:"id"`
	Score float64   `json:"score,string"`
}
