package executor

import (
	"database/sql/driver"
	"errors"
)

const (
	INTERVAL Kind = "INTERVAL"
	TASK     Kind = "TASK"
	CRON     Kind = "CRON"
)

type Kind string

func (k *Kind) Scan(value interface{}) error {
	asBytes, ok := value.(string)
	if !ok {
		return errors.New("scan source is not []byte")
	}
	*k = Kind(asBytes)
	return nil
}

func (k *Kind) Value() (driver.Value, error) {
	return string(*k), nil
}
