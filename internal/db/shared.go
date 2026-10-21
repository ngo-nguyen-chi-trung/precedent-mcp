package db

import (
	"reflect"
	"strings"

	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"github.com/weaviate/weaviate/entities/schema"

	"github.com/ngo/precedent_mcp/internal/env"
)

type Text2VecTransformersConfig struct {
	PoolingStrategy    string `json:"poolingStrategy"`
	VectorizeClassName bool   `json:"vectorizeClassName"`
}

func Text2VecTransformersModuleConfig() Text2VecTransformersConfig {
	return Text2VecTransformersConfig{
		PoolingStrategy:    "masked_mean",
		VectorizeClassName: false,
	}
}

type TagField struct {
	name         string
	dataType     string
	tokenization string
	nested       []TagField
}

func jsonTag(field reflect.StructField) string {
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

func weaviateTag(field reflect.StructField) string {
	dataType := field.Tag.Get("weaviate")
	if dataType == "-" {
		return ""
	}
	return dataType
}

func tokenizationTag(field reflect.StructField) string {
	tokenization := field.Tag.Get("tokenization")
	if tokenization == "-" {
		return ""
	}
	return tokenization
}

func nestedTag(field reflect.StructField) bool {
	_, ok := field.Tag.Lookup("nested")
	return ok
}

func TagFields(t reflect.Type) []TagField {
	var fields []TagField
	for i := range t.NumField() {
		field := t.Field(i)
		f := TagField{
			name:         jsonTag(field),
			dataType:     weaviateTag(field),
			tokenization: tokenizationTag(field),
		}
		if nestedTag(field) {
			f.nested = TagFields(field.Type)
		}
		fields = append(fields, f)
	}
	return fields
}

func Properties[T any]() []*models.Property {
	var props []*models.Property
	for _, f := range TagFields(reflect.TypeFor[T]()) {
		if f.dataType == "" {
			continue
		}
		props = append(props, &models.Property{
			Name:         f.name,
			DataType:     schema.DataType(f.dataType).PropString(),
			Tokenization: f.tokenization,
		})
	}
	return props
}

func SearchFields(fields []TagField) []graphql.Field {
	var out []graphql.Field
	for _, f := range fields {
		if f.name == "" {
			continue
		}
		out = append(out, graphql.Field{Name: f.name, Fields: SearchFields(f.nested)})
	}
	return out
}

func hybridSearch(query string) *graphql.HybridArgumentBuilder {
	return (&graphql.HybridArgumentBuilder{}).
		WithQuery(query).
		WithAlpha(float32(env.RankHybridAlpha)).
		WithFusionType(graphql.RelativeScore)
}
