package util

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSON json.RawMessage

var nullJSON = []byte(`null`)

var ErrJSONInvalid = errors.New("json_invalid")

func FromJSON(j json.RawMessage) JSON {
	return JSON(j)
}

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 || bytes.Equal(j, nullJSON) {
		return nil, nil
	}

	if err := json.Unmarshal(j, &json.RawMessage{}); err != nil {
		return []byte{}, err
	}

	return []byte(j), nil
}

func (j *JSON) Scan(src interface{}) error {
	var source []byte

	switch t := src.(type) {
	case string:
		if t == "" {
			source = nullJSON
		} else {
			source = []byte(t)
		}
	case []byte:
		if len(t) == 0 {
			source = nullJSON
		} else {
			source = t
		}
	case nil:
		*j = nullJSON
	default:
		return ErrJSONInvalid
	}

	*j = append((*j)[0:0], source...)

	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return nullJSON, nil
	}

	return j, nil
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return ErrJSONInvalid
	}

	*j = append((*j)[0:0], data...)

	return nil
}

func (j *JSON) Unmarshal(v interface{}) error {
	if len(*j) == 0 {
		*j = nullJSON
	}

	return json.Unmarshal([]byte(*j), v)
}
