package model

import (
	"encoding/json"
	"fmt"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ToolListResponse struct {
	*pb.ToolListResponse
}

func (r *ToolListResponse) FromToolEntries(entries []assetmodel.ToolEntry) error {
	tools := make([]*pb.Tool, 0, len(entries))
	for _, entry := range entries {
		input, err := json.Marshal(entry.InputSchema)
		if err != nil {
			return err
		}
		tool := &pb.Tool{Name: entry.Name, Description: entry.Description, InputSchema: input}
		if entry.OutputSchema != nil {
			if tool.OutputSchema, err = json.Marshal(entry.OutputSchema); err != nil {
				return err
			}
		}
		tools = append(tools, tool)
	}
	r.ToolListResponse = &pb.ToolListResponse{Tools: tools}
	return nil
}

func UnknownToolError(name env.ToolName) error {
	message := fmt.Sprintf("unknown tool: %s", name)
	st, err := status.New(codes.InvalidArgument, message).WithDetails(&pb.RPCError{
		Code:    int32(env.CodeInvalidParams),
		Message: message,
	})
	if err != nil {
		return err
	}
	return st.Err()
}

func InvalidArgsJSONError(err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInvalidArgument, Message: fmt.Sprintf("invalid args JSON: %v", err)}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}

func InvalidArgsSchemaError(err error) *pb.ToolCallResponse {
	data := Error{Code: env.ServiceErrorInvalidArgument, Message: fmt.Sprintf("invalid args: %v", err)}.JSON()
	return &pb.ToolCallResponse{Data: data, IsError: true}
}
