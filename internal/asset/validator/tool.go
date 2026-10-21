package validator

import (
	"bytes"
	"encoding/json"

	"github.com/ngo/precedent_mcp/internal/asset/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func InputSchemas() Step[[]model.ToolEntry] {
	return func(entries []model.ToolEntry) error {
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		for i := range entries {
			input, err := json.Marshal(entries[i].InputSchema)
			if err != nil {
				return err
			}
			document, err := jsonschema.UnmarshalJSON(bytes.NewReader(input))
			if err != nil {
				return err
			}
			url := "tool:" + entries[i].Name
			if err := compiler.AddResource(url, document); err != nil {
				return err
			}
			schema, err := compiler.Compile(url)
			if err != nil {
				return err
			}
			entries[i].Compiled = schema
		}
		return nil
	}
}
