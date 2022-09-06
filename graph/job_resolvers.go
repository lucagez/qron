package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/lucagez/tinyq/executor"
	"strconv"

	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
)

// CreateJob is the resolver for the createJob field.
func (r *mutationResolver) CreateJob(ctx context.Context, args *model.CreateHTTPJobArgs) (*sqlc.TinyJob, error) {
	config, _ := json.Marshal(executor.HttpConfig{
		Url:    args.URL,
		Method: args.Method,
	})
	job, err := r.Queries.CreateHttpJob(ctx, sqlc.CreateHttpJobParams{
		RunAt:  args.RunAt,
		Name:   sql.NullString{String: args.Name, Valid: true},
		State:  sql.NullString{String: args.State, Valid: true},
		Config: string(config),
	})
	return &job, err
}

// UpdateJobByName is the resolver for the updateJobByName field.
func (r *mutationResolver) UpdateJobByName(ctx context.Context, name string, args *model.UpdateHTTPJobArgs) (*sqlc.TinyJob, error) {
	config := executor.HttpConfig{}
	if args.URL != nil {
		config.Url = *args.URL
	}
	if args.Method != nil {
		config.Method = *args.Method
	}
	buf, _ := json.Marshal(config)
	params := sqlc.UpdateJobByNameParams{
		Name:   sql.NullString{String: name, Valid: true},
		Config: string(buf),
	}
	if args.RunAt != nil {
		params.RunAt = args.RunAt
	}
	if args.State != nil {
		params.State = args.State
	}
	job, err := r.Queries.UpdateJobByName(ctx, params)
	return &job, err
}

// UpdateJobByID is the resolver for the updateJobById field.
func (r *mutationResolver) UpdateJobByID(ctx context.Context, id string, args *model.UpdateHTTPJobArgs) (*sqlc.TinyJob, error) {
	i, _ := strconv.ParseInt(id, 10, 64)
	config := executor.HttpConfig{}
	if args.URL != nil {
		config.Url = *args.URL
	}
	if args.Method != nil {
		config.Method = *args.Method
	}
	buf, _ := json.Marshal(config)
	params := sqlc.UpdateJobByIDParams{
		ID:     i,
		Config: string(buf),
	}
	if args.RunAt != nil {
		params.RunAt = args.RunAt
	}
	if args.State != nil {
		params.State = args.State
	}
	job, err := r.Queries.UpdateJobByID(ctx, params)
	return &job, err
}

// DeleteJobByName is the resolver for the deleteJobByName field.
func (r *mutationResolver) DeleteJobByName(ctx context.Context, name string) (*sqlc.TinyJob, error) {
	job, err := r.Queries.DeleteJobByName(ctx, sql.NullString{String: name, Valid: true})
	return &job, err
}

// DeleteJobByID is the resolver for the deleteJobByID field.
func (r *mutationResolver) DeleteJobByID(ctx context.Context, id string) (*sqlc.TinyJob, error) {
	i, _ := strconv.ParseInt(id, 10, 64)
	job, err := r.Queries.DeleteJobByID(ctx, i)
	return &job, err
}

// SearchJobs is the resolver for the searchJobs field.
func (r *queryResolver) SearchJobs(ctx context.Context, args model.QueryJobsArgs) ([]*sqlc.TinyJob, error) {
	if args.Limit > 1000 {
		return nil, errors.New("requesting too many jobs")
	}
	jobs, err := r.Queries.SearchJobs(ctx, sqlc.SearchJobsParams{
		// Search term
		Query:  args.Filter,
		Offset: int32(args.Skip),
		Limit:  int32(args.Limit),
	})
	var refs []*sqlc.TinyJob
	for i := range jobs {
		refs = append(refs, &jobs[i])
	}
	return refs, err
}

// QueryJobByName is the resolver for the queryJobByName field.
func (r *queryResolver) QueryJobByName(ctx context.Context, name string) (*sqlc.TinyJob, error) {
	job, err := r.Queries.GetJobByName(ctx, sql.NullString{String: name, Valid: true})
	return &job, err
}

// QueryJobByID is the resolver for the queryJobByID field.
func (r *queryResolver) QueryJobByID(ctx context.Context, id string) (*sqlc.TinyJob, error) {
	i, _ := strconv.ParseInt(id, 10, 64)
	job, err := r.Queries.GetJobByID(ctx, i)
	return &job, err
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
