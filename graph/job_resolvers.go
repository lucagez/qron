package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.22

import (
	"context"
	"errors"
	"fmt"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	tinyctx "github.com/lucagez/tinyq/ctx"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
)

// CreateJob is the resolver for the createJob field.
func (r *mutationResolver) CreateJob(ctx context.Context, args model.CreateJobArgs) (sqlc.TinyJob, error) {
	var timeout int32
	if args.Timeout != nil {
		timeout = int32(*args.Timeout)
	}

	startAt := time.Now()
	if args.StartAt != nil {
		startAt = *args.StartAt
	}

	meta := []byte("{}")
	if args.Meta != nil {
		meta = []byte(*args.Meta)
	}

	return r.Queries.CreateJob(ctx, sqlc.CreateJobParams{
		Expr:     args.Expr,
		Name:     args.Name,
		State:    args.State,
		Executor: tinyctx.FromCtx(ctx),
		Timeout:  timeout,
		StartAt:  pgtype.Timestamptz{Time: startAt, Valid: true},
		Meta:     meta,
	})
}

// UpdateJobByName is the resolver for the updateJobByName field.
func (r *mutationResolver) UpdateJobByName(ctx context.Context, name string, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByNameParams{
		Name:     name,
		Executor: tinyctx.FromCtx(ctx),
	}
	if args.Expr != nil {
		params.Expr = args.Expr
	}
	if args.State != nil {
		params.State = args.State
	}
	if args.Timeout != nil {
		params.Timeout = args.Timeout
	}
	return r.Queries.UpdateJobByName(ctx, params)
}

// UpdateJobByID is the resolver for the updateJobById field.
func (r *mutationResolver) UpdateJobByID(ctx context.Context, id int64, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByIDParams{
		ID:       id,
		Executor: tinyctx.FromCtx(ctx),
	}
	if args.Expr != nil {
		params.Expr = args.Expr
	}
	if args.State != nil {
		params.State = args.State
	}
	if args.Timeout != nil {
		params.Timeout = args.Timeout
	}
	return r.Queries.UpdateJobByID(ctx, params)
}

// DeleteJobByName is the resolver for the deleteJobByName field.
func (r *mutationResolver) DeleteJobByName(ctx context.Context, name string) (sqlc.TinyJob, error) {
	return r.Queries.DeleteJobByName(ctx, sqlc.DeleteJobByNameParams{
		Name:     name,
		Executor: tinyctx.FromCtx(ctx),
	})
}

// DeleteJobByID is the resolver for the deleteJobByID field.
func (r *mutationResolver) DeleteJobByID(ctx context.Context, id int64) (sqlc.TinyJob, error) {
	return r.Queries.DeleteJobByID(ctx, sqlc.DeleteJobByIDParams{
		ID:       id,
		Executor: tinyctx.FromCtx(ctx),
	})
}

// FetchForProcessing is the resolver for the fetchForProcessing field.
func (r *mutationResolver) FetchForProcessing(ctx context.Context, limit int) ([]sqlc.TinyJob, error) {
	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	q := r.Queries.WithTx(tx)
	jobs, err := q.FetchDueJobs(ctx, sqlc.FetchDueJobsParams{
		Limit:    int32(limit),
		Executor: tinyctx.FromCtx(ctx),
	})
	if err != nil {
		tx.Rollback(ctx)
		return nil, err
	}

	if tx.Commit(ctx) != nil {
		return nil, err
	}

	return jobs, nil
}

// CommitJobs is the resolver for the commitJobs field.
func (r *mutationResolver) CommitJobs(ctx context.Context, commits []model.CommitArgs) ([]int64, error) {
	var batch []sqlc.BatchUpdateJobsParams
	for _, commit := range commits {
		var state string
		if commit.State != nil {
			state = *commit.State
		}

		var expr string
		if commit.Expr != nil {
			expr = *commit.Expr
		}

		batch = append(batch, sqlc.BatchUpdateJobsParams{
			ID:        commit.ID,
			LastRunAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			State:     state,
			Expr:      expr,
			Status:    sqlc.TinyStatusSUCCESS,
			Executor:  tinyctx.FromCtx(ctx),
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
func (r *mutationResolver) FailJobs(ctx context.Context, commits []model.CommitArgs) ([]int64, error) {
	var batch []sqlc.BatchUpdateJobsParams
	for _, commit := range commits {
		var state string
		if commit.State != nil {
			state = *commit.State
		}

		var expr string
		if commit.Expr != nil {
			expr = *commit.Expr
		}

		batch = append(batch, sqlc.BatchUpdateJobsParams{
			ID:        commit.ID,
			LastRunAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			State:     state,
			Expr:      expr,
			Status:    sqlc.TinyStatusFAILURE,
			Executor:  tinyctx.FromCtx(ctx),
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
func (r *mutationResolver) RetryJobs(ctx context.Context, commits []model.CommitArgs) ([]int64, error) {
	var batch []sqlc.BatchUpdateJobsParams
	for _, commit := range commits {
		var state string
		if commit.State != nil {
			state = *commit.State
		}

		var expr string
		if commit.Expr != nil {
			expr = *commit.Expr
		}

		batch = append(batch, sqlc.BatchUpdateJobsParams{
			ID:        commit.ID,
			LastRunAt: pgtype.Timestamptz{Valid: true, Time: time.Now()},
			State:     state,
			Expr:      expr,
			Status:    sqlc.TinyStatusREADY,
			Executor:  tinyctx.FromCtx(ctx),
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
		Executor: tinyctx.FromCtx(ctx),
	})
}

// QueryJobByName is the resolver for the queryJobByName field.
func (r *queryResolver) QueryJobByName(ctx context.Context, name string) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByName(ctx, sqlc.GetJobByNameParams{
		Name:     name,
		Executor: tinyctx.FromCtx(ctx),
	})
}

// QueryJobByID is the resolver for the queryJobByID field.
func (r *queryResolver) QueryJobByID(ctx context.Context, id int64) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByID(ctx, sqlc.GetJobByIDParams{
		ID:       id,
		Executor: tinyctx.FromCtx(ctx),
	})
}

// Name is the resolver for the Name field.
func (r *tinyJobResolver) Name(ctx context.Context, obj *sqlc.TinyJob) (*string, error) {
	return &obj.Name, nil
}

// RunAt is the resolver for the run_at field.
func (r *tinyJobResolver) RunAt(ctx context.Context, obj *sqlc.TinyJob) (time.Time, error) {
	return obj.RunAt.Time, nil
}

// LastRunAt is the resolver for the last_run_at field.
func (r *tinyJobResolver) LastRunAt(ctx context.Context, obj *sqlc.TinyJob) (*time.Time, error) {
	return &obj.LastRunAt.Time, nil
}

// Timeout is the resolver for the timeout field.
func (r *tinyJobResolver) Timeout(ctx context.Context, obj *sqlc.TinyJob) (*int, error) {
	timeout := int(obj.Timeout)
	return &timeout, nil
}

// Status is the resolver for the status field.
func (r *tinyJobResolver) Status(ctx context.Context, obj *sqlc.TinyJob) (string, error) {
	return string(obj.Status), nil
}

// Meta is the resolver for the meta field.
func (r *tinyJobResolver) Meta(ctx context.Context, obj *sqlc.TinyJob) (string, error) {
	panic(fmt.Errorf("not implemented: Meta - meta"))
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

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *tinyJobResolver) State(ctx context.Context, obj *sqlc.TinyJob) (*string, error) {
	// TODO: might panic
	return &obj.State, nil
}
