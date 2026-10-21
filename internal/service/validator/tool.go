package validator

import (
	"bytes"
	"context"
	"slices"

	pb "github.com/ngo/precedent_mcp/gen/proto/v1"
	"github.com/ngo/precedent_mcp/internal/asset"
	assetmodel "github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/ngo/precedent_mcp/internal/env"
	"github.com/ngo/precedent_mcp/internal/service/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func KnownTool[V any](registered map[env.ToolName]V, store *asset.ToolStore) Step[*model.ToolCallRequest, *pb.ToolCallResponse] {
	return func(_ context.Context, req *model.ToolCallRequest) (*pb.ToolCallResponse, error) {
		entries, err := store.Tools()
		if err != nil {
			return nil, err
		}
		i := slices.IndexFunc(entries, func(entry assetmodel.ToolEntry) bool { return entry.Name == req.GetName() })
		if _, ok := registered[env.ToolName(req.GetName())]; !ok || i < 0 {
			return nil, model.UnknownToolError(env.ToolName(req.GetName()))
		}
		req.Schema = entries[i].Compiled
		return nil, nil
	}
}

func Arguments() Step[*model.ToolCallRequest, *pb.ToolCallResponse] {
	return func(_ context.Context, req *model.ToolCallRequest) (*pb.ToolCallResponse, error) {
		if req.Schema == nil {
			return nil, model.UnknownToolError(env.ToolName(req.GetName()))
		}
		if len(req.GetArguments()) == 0 {
			req.Arguments = []byte(`{}`)
		}
		args, err := jsonschema.UnmarshalJSON(bytes.NewReader(req.GetArguments()))
		if err != nil {
			return model.InvalidArgsJSONError(err), nil
		}
		if err := req.Schema.Validate(args); err != nil {
			return model.InvalidArgsSchemaError(err), nil
		}
		return nil, nil
	}
}
