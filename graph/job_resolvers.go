package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.24

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
	"github.com/lucagez/tinyq/sqlc"
)

// ValidateExprFormat is the resolver for the validateExprFormat field.
func (r *mutationResolver) ValidateExprFormat(ctx context.Context, expr string) (bool, error) {
	return r.Queries.ValidateExprFormat(ctx, expr)
}

// CreateJob is the resolver for the createJob field.
func (r *mutationResolver) CreateJob(ctx context.Context, executor string, args model.CreateJobArgs) (sqlc.TinyJob, error) {
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

	var retries int32
	if args.Retries != nil {
		retries = int32(*args.Retries)
	}

	return r.Queries.CreateJob(ctx, sqlc.CreateJobParams{
		Expr:     args.Expr,
		Name:     args.Name,
		State:    args.State,
		Executor: executor,
		Timeout:  timeout,
		StartAt:  pgtype.Timestamptz{Time: startAt, Valid: true},
		Meta:     meta,
		Owner:    sqlc.FromCtx(ctx),
		Retries:  retries,
	})
}

// UpdateJobByName is the resolver for the updateJobByName field.
func (r *mutationResolver) UpdateJobByName(ctx context.Context, executor string, name string, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByNameParams{
		Name:     name,
		Executor: executor,
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
func (r *mutationResolver) UpdateJobByID(ctx context.Context, executor string, id int64, args model.UpdateJobArgs) (sqlc.TinyJob, error) {
	params := sqlc.UpdateJobByIDParams{
		ID:       id,
		Executor: executor,
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

// UpdateStateByID is the resolver for the updateStateByID field.
func (r *mutationResolver) UpdateStateByID(ctx context.Context, executor string, id int64, state string) (sqlc.TinyJob, error) {
	return r.Queries.UpdateStateByID(ctx, sqlc.UpdateStateByIDParams{
		ID:       id,
		Executor: executor,
		State:    state,
	})
}

// UpdateExprByID is the resolver for the updateExprByID field.
func (r *mutationResolver) UpdateExprByID(ctx context.Context, executor string, id int64, expr string) (sqlc.TinyJob, error) {
	return r.Queries.UpdateExprByID(ctx, sqlc.UpdateExprByIDParams{
		ID:       id,
		Executor: executor,
		Expr:     expr,
	})
}

// DeleteJobByName is the resolver for the deleteJobByName field.
func (r *mutationResolver) DeleteJobByName(ctx context.Context, executor string, name string) (sqlc.TinyJob, error) {
	return r.Queries.DeleteJobByName(ctx, sqlc.DeleteJobByNameParams{
		Name:     name,
		Executor: executor,
	})
}

// DeleteJobByID is the resolver for the deleteJobByID field.
func (r *mutationResolver) DeleteJobByID(ctx context.Context, executor string, id int64) (sqlc.TinyJob, error) {
	return r.Queries.DeleteJobByID(ctx, sqlc.DeleteJobByIDParams{
		ID:       id,
		Executor: executor,
	})
}

// StopJob is the resolver for the stopJob field.
func (r *mutationResolver) StopJob(ctx context.Context, executor string, id int64) (sqlc.TinyJob, error) {
	return r.Queries.StopJob(ctx, sqlc.StopJobParams{
		ID:       id,
		Executor: executor,
	})
}

// RestartJob is the resolver for the restartJob field.
func (r *mutationResolver) RestartJob(ctx context.Context, executor string, id int64) (sqlc.TinyJob, error) {
	return r.Queries.RestartJob(ctx, sqlc.RestartJobParams{
		ID:       id,
		Executor: executor,
	})
}

// FetchForProcessing is the resolver for the fetchForProcessing field.
func (r *mutationResolver) FetchForProcessing(ctx context.Context, executor string, limit int) ([]sqlc.TinyJob, error) {
	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	q := r.Queries.WithTx(tx)
	jobs, err := q.FetchDueJobs(ctx, sqlc.FetchDueJobsParams{
		Limit:    int32(limit),
		Executor: executor,
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
func (r *mutationResolver) CommitJobs(ctx context.Context, executor string, commits []model.CommitArgs) ([]int64, error) {
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
			ID:       commit.ID,
			State:    state,
			Expr:     expr,
			Status:   sqlc.TinyStatusSUCCESS,
			Executor: executor,
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
func (r *mutationResolver) FailJobs(ctx context.Context, executor string, commits []model.CommitArgs) ([]int64, error) {
	var batch []sqlc.BatchUpdateFailedJobsParams
	for _, commit := range commits {
		var state string
		if commit.State != nil {
			state = *commit.State
		}

		var expr string
		if commit.Expr != nil {
			expr = *commit.Expr
		}

		batch = append(batch, sqlc.BatchUpdateFailedJobsParams{
			ID:       commit.ID,
			State:    state,
			Expr:     expr,
			Executor: executor,
		})
	}

	// TODO: this does not ensure a job exists
	var failed []int64
	r.Queries.BatchUpdateFailedJobs(context.Background(), batch).Exec(func(i int, err error) {
		if err != nil {
			log.Println("error while updating failed jobs:", err)
			failed = append(failed, batch[i].ID)
		}
	})

	return failed, nil
}

// RetryJobs is the resolver for the retryJobs field.
func (r *mutationResolver) RetryJobs(ctx context.Context, executor string, commits []model.CommitArgs) ([]int64, error) {
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
			ID:       commit.ID,
			State:    state,
			Expr:     expr,
			Status:   sqlc.TinyStatusREADY,
			Executor: executor,
		})
	}

	// TODO: this does not ensure a job exists
	var failed []int64
	r.Queries.BatchUpdateJobs(ctx, batch).Exec(func(i int, err error) {
		if err != nil {
			failed = append(failed, batch[i].ID)
		}
	})

	return failed, nil
}

// SearchJobs is the resolver for the searchJobs field.
func (r *queryResolver) SearchJobs(ctx context.Context, executor string, args model.QueryJobsArgs) ([]sqlc.TinyJob, error) {
	if args.Limit > 1000 {
		return nil, errors.New("requesting too many jobs")
	}
	return r.Queries.SearchJobs(ctx, sqlc.SearchJobsParams{
		// Search term
		Query:    args.Filter,
		Offset:   int32(args.Skip),
		Limit:    int32(args.Limit),
		Executor: executor,
	})
}

// SearchJobsByMeta is the resolver for the searchJobsByMeta field.
func (r *queryResolver) SearchJobsByMeta(ctx context.Context, executor string, args model.QueryJobsMetaArgs) (model.SearchJobsByMetaResult, error) {
	statuses := strings.Join(args.Statuses, ",")

	var name string
	if args.Name != nil {
		name = *args.Name
	}

	rawquery := "{}"
	if args.Query != nil {
		rawquery = *args.Query
	}

	rows, err := r.Queries.SearchJobsByMeta(ctx, sqlc.SearchJobsByMetaParams{
		Query:     rawquery,
		Executor:  executor,
		Statuses:  statuses,
		From:      pgtype.Timestamptz{Time: args.From, Valid: true},
		To:        pgtype.Timestamptz{Time: args.To, Valid: true},
		Name:      name,
		Offset:    int32(args.Skip),
		Limit:     int32(args.Limit),
		IsOneShot: args.IsOneShot,
	})
	if err != nil {
		return model.SearchJobsByMetaResult{}, err
	}

	var jobs []sqlc.TinyJob
	for _, row := range rows {
		jobs = append(jobs, sqlc.TinyJob{
			ID:              row.ID,
			Name:            row.Name,
			Expr:            row.Expr,
			State:           row.State,
			Status:          row.Status,
			CreatedAt:       row.CreatedAt,
			LastRunAt:       row.LastRunAt,
			StartAt:         row.StartAt,
			RunAt:           row.RunAt,
			ExecutionAmount: row.ExecutionAmount,
			Retries:         row.Retries,
			Meta:            row.Meta,
			Timeout:         row.Timeout,
			Executor:        row.Executor,
			Owner:           row.Owner,
		})
	}

	total := 0
	if len(rows) > 0 {
		total = int(rows[0].TotalCount)
	}

	return model.SearchJobsByMetaResult{
		Jobs:  jobs,
		Total: total,
	}, nil
}

// QueryJobByName is the resolver for the queryJobByName field.
func (r *queryResolver) QueryJobByName(ctx context.Context, executor string, name string) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByName(ctx, sqlc.GetJobByNameParams{
		Name:     name,
		Executor: executor,
	})
}

// QueryJobByID is the resolver for the queryJobByID field.
func (r *queryResolver) QueryJobByID(ctx context.Context, executor string, id int64) (sqlc.TinyJob, error) {
	return r.Queries.GetJobByID(ctx, sqlc.GetJobByIDParams{
		ID:       id,
		Executor: executor,
	})
}

// RunAt is the resolver for the run_at field.
func (r *tinyJobResolver) RunAt(ctx context.Context, obj *sqlc.TinyJob) (time.Time, error) {
	return obj.RunAt.Time, nil
}

// LastRunAt is the resolver for the last_run_at field.
func (r *tinyJobResolver) LastRunAt(ctx context.Context, obj *sqlc.TinyJob) (*time.Time, error) {
	return &obj.LastRunAt.Time, nil
}

// StartAt is the resolver for the start_at field.
func (r *tinyJobResolver) StartAt(ctx context.Context, obj *sqlc.TinyJob) (*time.Time, error) {
	return &obj.StartAt.Time, nil
}

// CreatedAt is the resolver for the created_at field.
func (r *tinyJobResolver) CreatedAt(ctx context.Context, obj *sqlc.TinyJob) (time.Time, error) {
	return obj.CreatedAt.Time, nil
}

// Meta is the resolver for the meta field.
func (r *tinyJobResolver) Meta(ctx context.Context, obj *sqlc.TinyJob) (string, error) {
	return string(obj.Meta), nil
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
