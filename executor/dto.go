package executor

import (
	"time"

	"github.com/lucagez/tinyq/sqlc"
)

type TinyDto struct {
	ID              int64           `json:"id"`
	Expr            string          `json:"expr"`
	RunAt           time.Time       `json:"run_at,omitempty"`
	LastRunAt       time.Time       `json:"last_run_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	StartAt         time.Time       `json:"start_at"`
	ExecutionAmount int32           `json:"execution_amount"`
	Name            string          `json:"name"`
	Meta            string          `json:"meta"`
	Timeout         int32           `json:"timeout"`
	Status          sqlc.TinyStatus `json:"status"`
	State           string          `json:"state"`
	Executor        string          `json:"executor"`
}
