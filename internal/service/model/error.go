package model

import (
	"encoding/json"

	"github.com/ngo/precedent_mcp/internal/env"
)

type Error struct {
	Code    env.ServiceErrorCode `json:"code"`
	Message string               `json:"message"`
}

func (e Error) JSON() []byte {
	data, _ := json.Marshal(e)
	return data
}
