package db

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/fault"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate/entities/models"

	"github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
)

type DecisionStore interface {
	Create(ctx context.Context, d model.DecisionObject) error
	Get(ctx context.Context, id uuid.UUID) (*model.DecisionObject, error)
	Update(ctx context.Context, id uuid.UUID, d model.DecisionObject) error
	Search(ctx context.Context, query string, limit int) ([]model.DecisionSearch, error)
}

type DecisionRepo struct {
	client *weaviate.Client
	name   env.WeaviateClassName
}

func NewDecisionClass(ctx context.Context, client *weaviate.Client) error {
	exists, err := client.Schema().ClassExistenceChecker().
		WithClassName(string(env.ClassDecision)).
		Do(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	decisionClass := &models.Class{
		Class:      string(env.ClassDecision),
		Vectorizer: env.WeaviateText2VecModel.GetValue(),
		ModuleConfig: map[string]interface{}{
			env.WeaviateText2VecModel.GetValue(): Text2VecTransformersModuleConfig(),
		},
		Properties: Properties[model.DecisionObject](),
	}
	return client.Schema().ClassCreator().WithClass(decisionClass).Do(ctx)
}

func NewDecisionRepo(ctx context.Context, client *weaviate.Client) DecisionStore {
	err := NewDecisionClass(ctx, client)
	if err != nil {
		panic(err)
	}
	return &DecisionRepo{client, env.ClassDecision}
}

func (r *DecisionRepo) Create(ctx context.Context, d model.DecisionObject) error {
	_, err := r.client.Data().Creator().
		WithClassName(string(r.name)).
		WithID(d.ID.String()).
		WithProperties(d).
		Do(ctx)
	return err
}

func (r *DecisionRepo) Get(ctx context.Context, id uuid.UUID) (*model.DecisionObject, error) {
	objs, err := r.client.Data().ObjectsGetter().
		WithClassName(string(r.name)).
		WithID(id.String()).
		Do(ctx)
	var clientErr *fault.WeaviateClientError
	if errors.As(err, &clientErr) && clientErr.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(objs[0].Properties)
	if err != nil {
		return nil, err
	}
	var d model.DecisionObject
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DecisionRepo) Update(ctx context.Context, id uuid.UUID, d model.DecisionObject) error {
	return r.client.Data().Updater().
		WithClassName(string(r.name)).
		WithID(id.String()).
		WithProperties(d).
		WithMerge().
		Do(ctx)
}

func (r *DecisionRepo) Search(ctx context.Context, query string, limit int) ([]model.DecisionSearch, error) {
	where := filters.Where().
		WithPath([]string{"status"}).
		WithOperator(filters.NotEqual).
		WithValueText(string(env.StatusSuperseded))
	resp, err := r.client.GraphQL().Get().
		WithClassName(string(r.name)).
		WithHybrid(hybridSearch(query)).
		WithWhere(where).
		WithLimit(limit).
		WithFields(SearchFields(TagFields(reflect.TypeFor[model.DecisionSearch]()))...).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		errs := make([]error, 0, len(resp.Errors))
		for _, e := range resp.Errors {
			errs = append(errs, errors.New(e.Message))
		}
		return nil, errors.Join(errs...)
	}

	raw, err := json.Marshal(resp.Data["Get"])
	if err != nil {
		return nil, err
	}
	var results map[string][]model.DecisionSearch
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, err
	}
	return results[string(r.name)], nil
}
