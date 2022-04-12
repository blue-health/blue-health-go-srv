package util

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

// UUID can be used with the standard sql package to represent a
// UUID value that can be NULL in the database
type UUID struct {
	UUID  uuid.UUID
	Valid bool
}

func FromUUID(id uuid.UUID) UUID {
	return UUID{
		UUID:  id,
		Valid: true,
	}
}

func (u UUID) Interface() interface{} {
	if !u.Valid {
		return nil
	}

	return u.UUID
}

func NewUUID(u uuid.UUID) UUID {
	return UUID{UUID: u, Valid: true}
}

func (u UUID) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}

	return u.UUID.Value()
}

func (u *UUID) Scan(src interface{}) error {
	if src == nil {
		u.UUID, u.Valid = uuid.Nil, false
		return nil
	}

	u.Valid = true

	return u.UUID.Scan(src)
}

func (u UUID) MarshalJSON() ([]byte, error) {
	if u.Valid {
		return json.Marshal(u.UUID.String())
	}

	return json.Marshal(nil)
}

func (u *UUID) UnmarshalJSON(text []byte) error {
	u.Valid = false
	u.UUID = uuid.Nil

	if string(text) == "null" {
		return nil
	}

	s := string(text)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")

	us, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	u.UUID = us
	u.Valid = true

	return nil
}

// UnmarshalText will unmarshal text value into
// the propert representation of that value.
func (u *UUID) UnmarshalText(text []byte) error {
	return u.UnmarshalJSON(text)
}
