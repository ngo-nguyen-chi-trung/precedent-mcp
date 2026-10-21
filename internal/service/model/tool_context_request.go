package model

import (
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/ngo/precedent_mcp/internal/db/model"
)

type AddContextRequest struct {
	Entries []ContextEntry `json:"entries"`
}

func (r AddContextRequest) ToContextObjects() []dbmodel.ContextObject {
	now := time.Now().UTC()
	objects := make([]dbmodel.ContextObject, 0, len(r.Entries))
	for i, e := range r.Entries {
		objects = append(objects, dbmodel.ContextObject{
			ID:        uuid.NewSHA1(e.ThreadID, []byte(e.Context)),
			ThreadID:  e.ThreadID.String(),
			Content:   e.Context,
			CreatedAt: now.Add(time.Duration(i) * time.Millisecond),
		})
	}
	return objects
}

type ContextEntry struct {
	ThreadID uuid.UUID `json:"thread_id"`
	Context  string    `json:"context"`
}

type RetrieveContextRequest struct {
	Query string `json:"query"`
	TopN  *int   `json:"top_n"`
}

type GetContextThreadRequest struct {
	ThreadID uuid.UUID `json:"thread_id"`
}
