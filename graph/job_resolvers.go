package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"database/sql"
	"errors"
	"time"

	pgx "github.com/jackc/pgx/v4"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
)

// CreateJob is the resolver for the createJob field.
func (r *mutationResolver) CreateJob(ctx context.Context, args model.CreateJobArgs) (sqlc.TinyJob, error) {
	return r.Queries.CreateJob(ctx, sqlc.CreateJobParams{
		RunAt:    args.RunAt,
		Name:     sql.NullString{String: args.Name, Valid: true},
		State:    sql.NullString{String: args.State, Valid: true},
		Executor: executor.FromCtx(ctx),
	})
}

// UpdateJobByName is the resolver for the updateJobByName field.
func (r *mutationResolver) UpdateJobByName(ctx context.Context, name string, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByNameParams{
		Name:     sql.NullString{String: name, Valid: true},
		Executor: executor.FromCtx(ctx),
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
func (r *mutationResolver) UpdateJobByID(ctx context.Context, id int64, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByIDParams{
		ID:       id,
		Executor: executor.FromCtx(ctx),
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
		Executor: executor.FromCtx(ctx),
	})
}

// DeleteJobByID is the resolver for the deleteJobByID field.
func (r *mutationResolver) DeleteJobByID(ctx context.Context, id int64) (sqlc.TinyJob, error) {
	return r.Queries.DeleteJobByID(ctx, sqlc.DeleteJobByIDParams{
		ID:       id,
		Executor: executor.FromCtx(ctx),
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
		Executor: executor.FromCtx(ctx),
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
func (r *mutationResolver) CommitJobs(ctx context.Context, ids []int64) ([]int64, error) {
	var batch []sqlc.BatchUpdateJobsParams
	for _, id := range ids {
		batch = append(batch, sqlc.BatchUpdateJobsParams{
			ID: id,
			LastRunAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			Status:   sqlc.TinyStatusSUCCESS,
			Executor: executor.FromCtx(ctx),
		})
	}

	// TODO: this does not ensure a job exists
	var failed []int64
	r.Queries.BatchUpdateJobs(context.Background(), batch).Exec(func(i int, err error) {
		if err != nil {
			failed = append(failed, batch[i].ID)
		}
	})

	return failed, nil
}

// FailJobs is the resolver for the failJobs field.
func (r *mutationResolver) FailJobs(ctx context.Context, ids []int64) ([]int64, error) {
	var batch []sqlc.BatchUpdateJobsParams
	for _, id := range ids {
		batch = append(batch, sqlc.BatchUpdateJobsParams{
			ID: id,
			LastRunAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			Status:   sqlc.TinyStatusFAILURE,
			Executor: executor.FromCtx(ctx),
		})
	}

	// TODO: this does not ensure a job exists
	var failed []int64
	r.Queries.BatchUpdateJobs(context.Background(), batch).Exec(func(i int, err error) {
		if err != nil {
			failed = append(failed, batch[i].ID)
		}
	})

	return failed, nil
}

// RetryJobs is the resolver for the retryJobs field.
func (r *mutationResolver) RetryJobs(ctx context.Context, ids []int64) ([]int64, error) {
	var batch []sqlc.BatchUpdateJobsParams
	for _, id := range ids {
		batch = append(batch, sqlc.BatchUpdateJobsParams{
			ID: id,
			LastRunAt: sql.NullTime{
				Time:  time.Now(),
				Valid: true,
			},
			Status:   sqlc.TinyStatusREADY,
			Executor: executor.FromCtx(ctx),
		})
	}

	// TODO: this does not ensure a job exists
	var failed []int64
	r.Queries.BatchUpdateJobs(context.Background(), batch).Exec(func(i int, err error) {
		if err != nil {
			failed = append(failed, batch[i].ID)
		}
	})

	return failed, nil
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
		Executor: executor.FromCtx(ctx),
	})
}

// QueryJobByName is the resolver for the queryJobByName field.
func (r *queryResolver) QueryJobByName(ctx context.Context, name string) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByName(ctx, sqlc.GetJobByNameParams{
		Name:     sql.NullString{String: name, Valid: true},
		Executor: executor.FromCtx(ctx),
	})
}

// QueryJobByID is the resolver for the queryJobByID field.
func (r *queryResolver) QueryJobByID(ctx context.Context, id int64) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByID(ctx, sqlc.GetJobByIDParams{
		ID:       id,
		Executor: executor.FromCtx(ctx),
	})
}

// Name is the resolver for the Name field.
func (r *tinyJobResolver) Name(ctx context.Context, obj *sqlc.TinyJob) (*string, error) {
	// TODO: might panic
	return &obj.Name.String, nil
}

// LastRunAt is the resolver for the last_run_at field.
func (r *tinyJobResolver) LastRunAt(ctx context.Context, obj *sqlc.TinyJob) (*time.Time, error) {
	return &obj.LastRunAt.Time, nil
}

// State is the resolver for the state field.
func (r *tinyJobResolver) State(ctx context.Context, obj *sqlc.TinyJob) (*string, error) {
	// TODO: might panic
	return &obj.State.String, nil
}

// Status is the resolver for the status field.
func (r *tinyJobResolver) Status(ctx context.Context, obj *sqlc.TinyJob) (string, error) {
	return string(obj.Status), nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// TinyJob returns generated.TinyJobResolver implementation.
func (r *Resolver) TinyJob() generated.TinyJobResolver { return &tinyJobResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type tinyJobResolver struct{ *Resolver }
