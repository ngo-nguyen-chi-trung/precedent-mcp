package model

import (
	"time"

	"github.com/google/uuid"
)

type ContextObject struct {
	ID        uuid.UUID `json:"-"          weaviate:"-"    tokenization:"-"`
	ThreadID  string    `json:"thread_id"  weaviate:"text" tokenization:"field"`
	Content   string    `json:"content"    weaviate:"text" tokenization:"word"`
	CreatedAt time.Time `json:"created_at" weaviate:"date"`
}

type ContextSearch struct {
	ThreadID string          `json:"thread_id"   weaviate:"text" tokenization:"field"`
	Content  string          `json:"content"     weaviate:"text" tokenization:"word"`
	Metadata ContextMetadata `json:"_additional" nested:""`
}

type ContextMetadata struct {
	Score float64 `json:"score,string"`
}
