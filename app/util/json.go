package util

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type JSON json.RawMessage

var ErrInvalidJSON = errors.New("invalid_json")

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}

	return driver.Value([]byte(j)), nil
}

func (j *JSON) Scan(src interface{}) error {
	asBytes, ok := src.([]byte)
	if !ok {
		return ErrInvalidJSON
	}

	if err := json.Unmarshal(asBytes, &j); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return nil
}

func (j *JSON) MarshalJSON() ([]byte, error) {
	return *j, nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return ErrInvalidJSON
	}

	*j = append((*j)[0:0], data...)

	return nil
}
