package executor

import (
	"database/sql"
	"time"
)

type Job struct {
	Id              int            `json:"id" db:"id"`
	Status          Status         `json:"status" db:"status"`
	LastRunAt       sql.NullTime   `json:"last_run_at" db:"last_run_at"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	RunAt           string         `json:"run_at" db:"run_at"`
	Name            sql.NullString `json:"name" db:"name"`
	ExecutionAmount int            `json:"execution_amount" db:"execution_amount"`
	Timeout         int            `json:"timeout" db:"timeout"`
	State           string         `json:"state" db:"state"`
	Config          string         `json:"config" db:"config"`
	Kind            Kind           `json:"kind" db:"kind"`
	ExecutorType    string         `json:"executor" db:"executor"`
}
