package db

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"

	"github.com/ngo/precedent_mcp/internal/db/model"
	"github.com/ngo/precedent_mcp/internal/env"
)

type ContextStore interface {
	Create(ctx context.Context, objects []model.ContextObject) (succeeded, failed []uuid.UUID, err error)
	Search(ctx context.Context, query string, limit int) ([]model.ContextSearch, error)
	Get(ctx context.Context, threadID string, limit int) ([]model.ContextObject, error)
}

type ContextRepo struct {
	client *weaviate.Client
	name   env.WeaviateClassName
}

func NewContextClass(ctx context.Context, client *weaviate.Client) error {
	exists, err := client.Schema().ClassExistenceChecker().
		WithClassName(string(env.ClassContext)).
		Do(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	contextClass := &models.Class{
		Class:      string(env.ClassContext),
		Vectorizer: env.WeaviateText2VecModel.GetValue(),
		ModuleConfig: map[string]interface{}{
			env.WeaviateText2VecModel.GetValue(): Text2VecTransformersModuleConfig(),
		},
		Properties: Properties[model.ContextObject](),
	}
	return client.Schema().ClassCreator().WithClass(contextClass).Do(ctx)
}

func NewContextRepo(ctx context.Context, client *weaviate.Client) ContextStore {
	err := NewContextClass(ctx, client)
	if err != nil {
		panic(err)
	}
	return &ContextRepo{client, env.ClassContext}
}

func (r *ContextRepo) Create(ctx context.Context, objects []model.ContextObject) (succeeded, failed []uuid.UUID, err error) {
	batch := make([]*models.Object, 0, len(objects))
	for _, o := range objects {
		batch = append(batch, &models.Object{
			Class:      string(r.name),
			ID:         strfmt.UUID(o.ID.String()),
			Properties: o,
		})
	}
	resp, err := r.client.Batch().ObjectsBatcher().WithObjects(batch...).Do(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, o := range resp {
		id, err := uuid.Parse(o.ID.String())
		if err != nil {
			continue
		}
		if o.Result != nil && o.Result.Errors != nil && len(o.Result.Errors.Error) > 0 {
			failed = append(failed, id)
			continue
		}
		succeeded = append(succeeded, id)
	}
	return succeeded, failed, nil
}

func (r *ContextRepo) Search(ctx context.Context, query string, limit int) ([]model.ContextSearch, error) {
	resp, err := r.client.GraphQL().Get().
		WithClassName(string(r.name)).
		WithHybrid(hybridSearch(query)).
		WithLimit(limit).
		WithFields(SearchFields(TagFields(reflect.TypeFor[model.ContextSearch]()))...).
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
	var results map[string][]model.ContextSearch
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, err
	}
	return results[string(r.name)], nil
}

func (r *ContextRepo) Get(ctx context.Context, threadID string, limit int) ([]model.ContextObject, error) {
	where := filters.Where().
		WithPath([]string{"thread_id"}).
		WithOperator(filters.Equal).
		WithValueText(threadID)
	resp, err := r.client.GraphQL().Get().
		WithClassName(string(r.name)).
		WithWhere(where).
		WithSort(graphql.Sort{Path: []string{"created_at"}, Order: graphql.Desc}).
		WithLimit(limit).
		WithFields(SearchFields(TagFields(reflect.TypeFor[model.ContextObject]()))...).
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
	var results map[string][]model.ContextObject
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, err
	}
	return results[string(r.name)], nil
}
