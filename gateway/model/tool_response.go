package model

import (
	"encoding/json"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/env"
)

type Tool struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	InputSchema  json.RawMessage `json:"inputSchema"`
	OutputSchema json.RawMessage `json:"outputSchema,omitempty"`
}

type ToolsListResult struct {
	ResultBase
	CacheHints
	Tools []Tool `json:"tools"`
}

func (r *Response) FromToolListResponse(res *pb.ToolListResponse) {
	tools := make([]Tool, 0, len(res.GetTools()))
	for _, tool := range res.GetTools() {
		tools = append(tools, Tool{
			Name:         tool.GetName(),
			Description:  tool.GetDescription(),
			InputSchema:  tool.GetInputSchema(),
			OutputSchema: tool.GetOutputSchema(),
		})
	}
	r.Result = ToolsListResult{ResultBase: complete, CacheHints: publicCache, Tools: tools}
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CallToolResult struct {
	ResultBase
	Content           []TextContent   `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent"`
	IsError           bool            `json:"isError"`
}

func (r *Response) FromToolCallResponse(res *pb.ToolCallResponse) {
	r.Result = CallToolResult{
		ResultBase:        complete,
		Content:           []TextContent{{Type: string(env.ContentText), Text: string(res.GetData())}},
		StructuredContent: res.GetData(),
		IsError:           res.GetIsError(),
	}
}
