package util

import (
	"database/sql/driver"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type UUIDArray []uuid.UUID

var ErrUUIDArrayInvalid = errors.New("uuid_array_invalid")

func (a UUIDArray) Value() (driver.Value, error) {
	s := make(pq.StringArray, len(a))

	for i := range a {
		s[i] = a[i].String()
	}

	return s.Value()
}

func (a UUIDArray) Contains(id uuid.UUID) bool {
	for _, i := range a {
		if i == id {
			return true
		}
	}

	return false
}

func (a *UUIDArray) Scan(src interface{}) error {
	var s pq.StringArray

	if err := s.Scan(src); err != nil {
		return err
	}

	*a = make(UUIDArray, len(s))

	for i := range s {
		u, err := uuid.Parse(s[i])
		if err != nil {
			return err
		}

		(*a)[i] = u
	}

	return nil
}
