package executor

import (
	"database/sql/driver"
	"errors"
)

const (
	READY   Status = "READY"
	PENDING Status = "PENDING"
	FAILURE Status = "FAILURE"
	SUCCESS Status = "SUCCESS"
)

type Status string

func (s *Status) Scan(value interface{}) error {
	asBytes, ok := value.(string)
	if !ok {
		return errors.New("scan source is not []byte")
	}
	*s = Status(asBytes)
	return nil
}

func (s *Status) Value() (driver.Value, error) {
	return string(*s), nil
}
