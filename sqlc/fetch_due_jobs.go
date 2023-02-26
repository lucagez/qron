package sqlc

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FetchDueJobsPartitioned modiefies the original query to fetch
// from today's partition
func (q *Queries) FetchDueJobsPartitioned(ctx context.Context, arg FetchDueJobsParams) ([]TinyJob, error) {
	today := time.Now().Format("2006-01-02")
	currentPartition := fmt.Sprintf("tiny.job_%s", today)
	query := strings.ReplaceAll(fetchDueJobs, "tiny.job", currentPartition)
	rows, err := q.db.Query(ctx, query, arg.Limit, arg.Executor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []TinyJob
	for rows.Next() {
		var i TinyJob
		if err := rows.Scan(
			&i.ID,
			&i.Expr,
			&i.RunAt,
			&i.LastRunAt,
			&i.CreatedAt,
			&i.StartAt,
			&i.ExecutionAmount,
			&i.Retries,
			&i.Name,
			&i.Meta,
			&i.Timeout,
			&i.Status,
			&i.State,
			&i.Executor,
			&i.Owner,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// [] partition query should work exactly like unpartitioned one
// [] add test for partitioned query
// [] partitioned query should pass tests also for unpartitioned query
// [] create function for creating new partition
// [] create job executor for creating new partition
