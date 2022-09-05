package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/lucagez/tinyq/executor"
	"github.com/lucagez/tinyq/graph/generated"
	"github.com/lucagez/tinyq/graph/model"
)

// CreateJob is the resolver for the createJob field.
func (r *mutationResolver) CreateJob(ctx context.Context, args *model.HTTPJobArgs) (*model.Job, error) {
	config, _ := json.Marshal(map[string]string{
		"url":    args.URL,
		"method": args.Method,
	})
	rows, err := r.Db.Query(ctx, `
		insert into tiny.job(run_at, name, state, config, status, executor)
		values (
		    $1,
		    $2,
		    $3,
		    $4,
		    'READY',
		    'HTTP'
		)
		returning *;
	`, args.RunAt, args.Name, args.State, string(config))
	if err != nil {
		return nil, err
	}

	var job executor.Job
	err = pgxscan.ScanOne(&job, rows)
	if err != nil {
		return nil, err
	}

	return &model.Job{
		ID:        string(rune(job.Id)),
		Status:    model.Status(job.Status),
		LastRunAt: job.LastRunAt,
		CreatedAt: job.CreatedAt,
		RunAt:     job.RunAt,
		Name:      job.Name,
		State:     job.State,
		Config:    job.Config,
	}, nil
}

// UpdateJobByName is the resolver for the updateJobByName field.
func (r *mutationResolver) UpdateJobByName(ctx context.Context, name string, args *model.HTTPJobArgs) (*model.Job, error) {
	config, _ := json.Marshal(map[string]string{
		"url":    args.URL,
		"method": args.Method,
	})
	rows, err := r.Db.Query(ctx, `
		update tiny.job
		set run_at = coalesce(nullif($2, ''), run_at),
			state = coalesce(nullif($3, ''), state),
			config = coalesce(nullif($4, ''), config)
		where name = $1
		returning *;
	`, name, args.RunAt, args.State, string(config))
	if err != nil {
		return nil, err
	}

	var job executor.Job
	err = pgxscan.ScanOne(&job, rows)
	if err != nil {
		return nil, err
	}

	return &model.Job{
		ID:        string(rune(job.Id)),
		Status:    model.Status(job.Status),
		LastRunAt: job.LastRunAt,
		CreatedAt: job.CreatedAt,
		RunAt:     job.RunAt,
		Name:      job.Name,
		State:     job.State,
		Config:    job.Config,
	}, nil
}

// UpdateJobByID is the resolver for the updateJobById field.
func (r *mutationResolver) UpdateJobByID(ctx context.Context, id string, args *model.HTTPJobArgs) (*model.Job, error) {
	config, _ := json.Marshal(map[string]string{
		"url":    args.URL,
		"method": args.Method,
	})
	rows, err := r.Db.Query(ctx, `
		update tiny.job
		set run_at = coalesce(nullif($2, ''), run_at),
			state = coalesce(nullif($3, ''), state),
			config = coalesce(nullif($4, ''), config)
		where id = $1
		returning *;
	`, id, args.RunAt, args.State, string(config))
	if err != nil {
		return nil, err
	}

	var job executor.Job
	err = pgxscan.ScanOne(&job, rows)
	if err != nil {
		return nil, err
	}

	return &model.Job{
		ID:        string(rune(job.Id)),
		Status:    model.Status(job.Status),
		LastRunAt: job.LastRunAt,
		CreatedAt: job.CreatedAt,
		RunAt:     job.RunAt,
		Name:      job.Name,
		State:     job.State,
		Config:    job.Config,
	}, nil
}

// DeleteJobByName is the resolver for the deleteJobByName field.
func (r *mutationResolver) DeleteJobByName(ctx context.Context, name string) (*model.Job, error) {
	panic(fmt.Errorf("not implemented: DeleteJobByName - deleteJobByName"))
}

// DeleteJobByID is the resolver for the deleteJobByID field.
func (r *mutationResolver) DeleteJobByID(ctx context.Context, id string) (*model.Job, error) {
	panic(fmt.Errorf("not implemented: DeleteJobByID - deleteJobByID"))
}

// SearchJobs is the resolver for the searchJobs field.
func (r *queryResolver) SearchJobs(ctx context.Context, args model.QueryJobsArgs) (*model.Job, error) {
	panic(fmt.Errorf("not implemented: SearchJobs - searchJobs"))
}

// QueryJobByName is the resolver for the queryJobByName field.
func (r *queryResolver) QueryJobByName(ctx context.Context, name string) (*model.Job, error) {
	panic(fmt.Errorf("not implemented: QueryJobByName - queryJobByName"))
}

// QueryJobByID is the resolver for the queryJobByID field.
func (r *queryResolver) QueryJobByID(ctx context.Context, id string) (*model.Job, error) {
	panic(fmt.Errorf("not implemented: QueryJobByID - queryJobByID"))
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }

// !!! WARNING !!!
// The code below was going to be deleted when updating resolvers. It has been copied here so you have
// one last chance to move it out of harms way if you want. There are two reasons this happens:
//   - When renaming or deleting a resolver the old code will be put in here. You can safely delete
//     it when you're done.
//   - You have helper methods in this file. Move them out to keep these resolver files clean.
func (r *mutationResolver) UpdateJob(ctx context.Context, args *model.HTTPJobArgs) (*model.Job, error) {
	panic(fmt.Errorf("not implemented: UpdateJob - updateJob"))
}
