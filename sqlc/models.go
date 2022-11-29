// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0

package sqlc

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

type TinyStatus string

const (
	TinyStatusREADY   TinyStatus = "READY"
	TinyStatusPENDING TinyStatus = "PENDING"
	TinyStatusFAILURE TinyStatus = "FAILURE"
	TinyStatusSUCCESS TinyStatus = "SUCCESS"
)

func (e *TinyStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TinyStatus(s)
	case string:
		*e = TinyStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for TinyStatus: %T", src)
	}
	return nil
}

type NullTinyStatus struct {
	TinyStatus TinyStatus
	Valid      bool // Valid is true if TinyStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullTinyStatus) Scan(value interface{}) error {
	if value == nil {
		ns.TinyStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.TinyStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullTinyStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.TinyStatus, nil
}

type TinyJob struct {
	ID              int64
	Expr            string
	RunAt           sql.NullTime
	LastRunAt       sql.NullTime
	CreatedAt       time.Time
	StartAt         time.Time
	ExecutionAmount int32
	Name            sql.NullString
	Timeout         sql.NullInt32
	Status          TinyStatus
	State           sql.NullString
	Executor        string
}
