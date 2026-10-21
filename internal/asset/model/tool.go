package model

import "github.com/santhosh-tekuri/jsonschema/v6"

type ToolSchema struct {
	Type                 string         `yaml:"type" json:"type"`
	Properties           map[string]any `yaml:"properties" json:"properties"`
	Required             []string       `yaml:"required,omitempty" json:"required,omitempty"`
	AdditionalProperties *bool          `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
}

type OutputSchema struct {
	Type  string       `yaml:"type" json:"type"`
	AnyOf []ToolSchema `yaml:"anyOf" json:"anyOf"`
}

type ToolEntry struct {
	Name         string             `yaml:"name"`
	Description  string             `yaml:"description"`
	InputSchema  ToolSchema         `yaml:"input_schema"`
	OutputSchema *OutputSchema      `yaml:"output_schema"`
	Compiled     *jsonschema.Schema `yaml:"-"`
}

type ToolList struct {
	Tools []ToolEntry `yaml:"tools"`
}
