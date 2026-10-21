// Test plan: shared.go
//
// Logic flow
// | Function                         | Step | Action                                                                                                            | Effect                                                           |
// |----------------------------------|------|-------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------|
// | Text2VecTransformersModuleConfig | 1    | Return the vectorizer module settings                                                                             | Pooling masked_mean; class name not vectorized                   |
// | TagFields                        | 1    | Per field in declaration order: json name up to the first comma, weaviate data type, tokenization; "-" becomes "" | Options such as omitempty and string are not part of a name      |
// | TagFields                        | 2    | Field with a nested tag → describe its fields one level down                                                      | _additional carries its own sub fields                           |
// | Properties                       | 1    | Keep the fields that have a Weaviate data type                                                                    | ID (all tags "-") is not a class property                        |
// | Properties                       | 2    | Build models.Property with name, data type, tokenization                                                          | Weaviate class schema in field order                             |
// | SearchFields                     | 1    | Keep the fields that have a JSON name                                                                             | ID (json "-") is not requested; search reads it from _additional |
// | SearchFields                     | 2    | Recurse into nested fields                                                                                        | _additional requests its sub fields                              |
// | hybridSearch                     | 1    | Hybrid query: the text, alpha RankHybridAlpha, relative score fusion                                              | Argument builder for Get().WithHybrid                            |
//
// Scenarios
// | Number | Function                         | Result  | Value class    | Input                                                             | Expected                                                                                                                                                                           |
// |--------|----------------------------------|---------|----------------|-------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
// | 1      | Text2VecTransformersModuleConfig | Success | Middle value   | None                                                              | PoolingStrategy masked_mean; VectorizeClassName false (step 1)                                                                                                                     |
// | 2      | Properties                       | Success | Middle value   | ContextObject                                                     | thread_id text field, content text word, created_at date; ID absent (steps 1, 2)                                                                                                   |
// | 3      | Properties                       | Success | Middle value   | DecisionObject                                                    | title text word, status text field, tags text[] field, supersedes text field, supersededBy text field, content text word, review text word, createdAt date; ID absent (steps 1, 2) |
// | 4      | Properties                       | Success | Boundary value | Struct whose only field is an ID with all tags "-"                | No properties (step 1)                                                                                                                                                             |
// | 5      | SearchFields                     | Success | Middle value   | TagFields(ContextSearch)                                          | thread_id, content, _additional with score (steps 1, 2)                                                                                                                            |
// | 6      | SearchFields                     | Success | Middle value   | TagFields(DecisionSearch)                                         | title, status, tags, supersedes, supersededBy, content, review, createdAt, _additional with id and score (steps 1, 2)                                                              |
// | 7      | SearchFields                     | Success | Middle value   | TagFields(ContextObject); the thread read                         | thread_id, content, created_at; no sub fields; ID absent (step 1)                                                                                                                  |
// | 8      | SearchFields                     | Success | Boundary value | TagFields of a struct whose only field is an ID with all tags "-" | No fields (step 1)                                                                                                                                                                 |
//
// Failure scenarios: none. shared.go returns values only; struct tags are fixed at compile time.
//
// Outside the db tag convention: a field without json, weaviate and tokenization tags; every db struct field declares all three.
//
// Branch coverage
// | Function     | Condition                      | True   | False        |
// |--------------|--------------------------------|--------|--------------|
// | Properties   | Field has a Weaviate data type | 2, 3   | 2, 3 (ID), 4 |
// | SearchFields | Field has a JSON name          | 5 to 7 | 7 (ID), 8    |
// | SearchFields | Field is nested                | 5, 6   | 7            |
//
// Test mapping
// | Test                                                                        | Scenarios                                                                                                                      |
// |-----------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------|
// | TestText2VecTransformersModuleConfig/masked mean pooling without class name | 1                                                                                                                              |
// | TestProperties/context object                                               | 2                                                                                                                              |
// | TestProperties/decision object                                              | 3                                                                                                                              |
// | TestProperties/only an excluded identifier                                  | 4                                                                                                                              |
// | TestSearchFields/context search                                             | 5                                                                                                                              |
// | TestSearchFields/decision search                                            | 6                                                                                                                              |
// | TestSearchFields/context object for the thread read                         | 7                                                                                                                              |
// | TestSearchFields/only an excluded identifier                                | 8                                                                                                                              |
// | No test                                                                     | hybridSearch: settings live in unexported builder fields and render only inside the Weaviate client; e2e search tests cover it |

package db_test

import . "github.com/ngo/precedent_mcp/internal/db"

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

type excludedIdentifierOnly struct {
	ID uuid.UUID `json:"-" weaviate:"-" tokenization:"-"`
}

func TestText2VecTransformersModuleConfig(t *testing.T) {
	tests := []struct {
		name string
		want Text2VecTransformersConfig
	}{
		{"masked mean pooling without class name", Text2VecTransformersConfig{PoolingStrategy: "masked_mean", VectorizeClassName: false}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Text2VecTransformersModuleConfig()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestProperties(t *testing.T) {
	tests := []struct {
		name            string
		givenProperties func() []*models.Property
		want            []*models.Property
	}{
		{
			name:            "context object",
			givenProperties: Properties[model.ContextObject],
			want: []*models.Property{
				{Name: "thread_id", DataType: []string{"text"}, Tokenization: "field"},
				{Name: "content", DataType: []string{"text"}, Tokenization: "word"},
				{Name: "created_at", DataType: []string{"date"}},
			},
		},
		{
			name:            "decision object",
			givenProperties: Properties[model.DecisionObject],
			want: []*models.Property{
				{Name: "title", DataType: []string{"text"}, Tokenization: "word"},
				{Name: "status", DataType: []string{"text"}, Tokenization: "field"},
				{Name: "tags", DataType: []string{"text[]"}, Tokenization: "field"},
				{Name: "supersedes", DataType: []string{"text"}, Tokenization: "field"},
				{Name: "supersededBy", DataType: []string{"text"}, Tokenization: "field"},
				{Name: "content", DataType: []string{"text"}, Tokenization: "word"},
				{Name: "review", DataType: []string{"text"}, Tokenization: "word"},
				{Name: "createdAt", DataType: []string{"date"}},
			},
		},
		{
			name:            "only an excluded identifier",
			givenProperties: Properties[excludedIdentifierOnly],
			want:            nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.givenProperties()

			assert.Equal(t, test.want, got)
		})
	}
}

func TestSearchFields(t *testing.T) {
	tests := []struct {
		name      string
		givenType reflect.Type
		want      []graphql.Field
	}{
		{
			name:      "context search",
			givenType: reflect.TypeFor[model.ContextSearch](),
			want: []graphql.Field{
				{Name: "thread_id"},
				{Name: "content"},
				{Name: "_additional", Fields: []graphql.Field{{Name: "score"}}},
			},
		},
		{
			name:      "decision search",
			givenType: reflect.TypeFor[model.DecisionSearch](),
			want: []graphql.Field{
				{Name: "title"},
				{Name: "status"},
				{Name: "tags"},
				{Name: "supersedes"},
				{Name: "supersededBy"},
				{Name: "content"},
				{Name: "review"},
				{Name: "createdAt"},
				{Name: "_additional", Fields: []graphql.Field{{Name: "id"}, {Name: "score"}}},
			},
		},
		{
			name:      "context object for the thread read",
			givenType: reflect.TypeFor[model.ContextObject](),
			want:      []graphql.Field{{Name: "thread_id"}, {Name: "content"}, {Name: "created_at"}},
		},
		{
			name:      "only an excluded identifier",
			givenType: reflect.TypeFor[excludedIdentifierOnly](),
			want:      nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SearchFields(TagFields(test.givenType))

			assert.Equal(t, test.want, got)
		})
	}
}
