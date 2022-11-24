package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	pgx "github.com/jackc/pgx/v4"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
)

// CreateJob is the resolver for the createJob field.
func (r *mutationResolver) CreateJob(ctx context.Context, args *model.CreateJobArgs) (sqlc.TinyJob, error) {
	return r.Queries.CreateJob(ctx, sqlc.CreateJobParams{
		RunAt:    args.RunAt,
		Name:     sql.NullString{String: args.Name, Valid: true},
		State:    sql.NullString{String: args.State, Valid: true},
		Executor: "TODO",
	})
}

// UpdateJobByName is the resolver for the updateJobByName field.
func (r *mutationResolver) UpdateJobByName(ctx context.Context, name string, args *model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByNameParams{
		Name:     sql.NullString{String: name, Valid: true},
		Executor: "TODO",
	}
	if args.RunAt != nil {
		params.RunAt = args.RunAt
	}
	if args.State != nil {
		params.State = args.State
	}
	return r.Queries.UpdateJobByName(ctx, params)
}

// UpdateJobByID is the resolver for the updateJobById field.
func (r *mutationResolver) UpdateJobByID(ctx context.Context, id string, args *model.UpdateJobArgs) (sqlc.TinyJob, error) {
	i, _ := strconv.ParseInt(id, 10, 64)
	params := sqlc.UpdateJobByIDParams{
		ID:       i,
		Executor: "TODO",
	}
	if args.RunAt != nil {
		params.RunAt = args.RunAt
	}
	if args.State != nil {
		params.State = args.State
	}
	return r.Queries.UpdateJobByID(ctx, params)
}

// DeleteJobByName is the resolver for the deleteJobByName field.
func (r *mutationResolver) DeleteJobByName(ctx context.Context, name string) (sqlc.TinyJob, error) {
	return r.Queries.DeleteJobByName(ctx, sqlc.DeleteJobByNameParams{
		Name:     sql.NullString{String: name, Valid: true},
		Executor: "TODO",
	})
}

// DeleteJobByID is the resolver for the deleteJobByID field.
func (r *mutationResolver) DeleteJobByID(ctx context.Context, id string) (sqlc.TinyJob, error) {
	i, _ := strconv.ParseInt(id, 10, 64)
	return r.Queries.DeleteJobByID(ctx, sqlc.DeleteJobByIDParams{
		ID:       i,
		Executor: "TODO",
	})
}

// FetchForProcessing is the resolver for the fetchForProcessing field.
func (r *mutationResolver) FetchForProcessing(ctx context.Context, limit int) ([]sqlc.TinyJob, error) {
	tx, err := r.DB.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	q := r.Queries.WithTx(tx)
	jobs, err := q.FetchDueJobs(context.Background(), sqlc.FetchDueJobsParams{
		Limit:    int32(limit),
		Executor: "TODO",
	})
	if err != nil {
		tx.Rollback(context.Background())
		return nil, err
	}

	if tx.Commit(context.Background()) != nil {
		return nil, err
	}

	return jobs, nil
}

// CommitJobs is the resolver for the commitJobs field.
func (r *mutationResolver) CommitJobs(ctx context.Context, jobs []string) ([]sqlc.TinyJob, error) {
	panic(fmt.Errorf("not implemented: CommitJobs - commitJobs"))
}

// FailJobs is the resolver for the failJobs field.
func (r *mutationResolver) FailJobs(ctx context.Context, jobs []string) ([]sqlc.TinyJob, error) {
	panic(fmt.Errorf("not implemented: FailJobs - failJobs"))
}

// RetryJobs is the resolver for the retryJobs field.
func (r *mutationResolver) RetryJobs(ctx context.Context, jobs []string) ([]sqlc.TinyJob, error) {
	panic(fmt.Errorf("not implemented: RetryJobs - retryJobs"))
}

// SearchJobs is the resolver for the searchJobs field.
func (r *queryResolver) SearchJobs(ctx context.Context, args model.QueryJobsArgs) ([]sqlc.TinyJob, error) {
	if args.Limit > 1000 {
		return nil, errors.New("requesting too many jobs")
	}
	return r.Queries.SearchJobs(ctx, sqlc.SearchJobsParams{
		// Search term
		Query:    args.Filter,
		Offset:   int32(args.Skip),
		Limit:    int32(args.Limit),
		Executor: "TODO",
	})
}

// QueryJobByName is the resolver for the queryJobByName field.
func (r *queryResolver) QueryJobByName(ctx context.Context, name string) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByName(ctx, sqlc.GetJobByNameParams{
		Name:     sql.NullString{String: name, Valid: true},
		Executor: "TODO",
	})
}

// QueryJobByID is the resolver for the queryJobByID field.
func (r *queryResolver) QueryJobByID(ctx context.Context, id string) (sqlc.TinyJob, error) {
	i, _ := strconv.ParseInt(id, 10, 64)
	return r.Queries.GetJobByID(ctx, sqlc.GetJobByIDParams{
		ID:       i,
		Executor: "TODO",
	})
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
