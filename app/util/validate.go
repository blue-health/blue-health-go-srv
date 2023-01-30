package util

import (
	"database/sql/driver"
	"reflect"

	"github.com/go-playground/validator/v10"
	"gopkg.in/guregu/null.v4"
)

var Validate = validator.New()

func init() {
	Validate.RegisterCustomTypeFunc(validateNullType,
		null.String{},
		null.Time{},
		null.Int{},
		null.Float{},
		null.Bool{},
	)
}

func validateNullType(field reflect.Value) interface{} {
	switch t := field.Interface().(type) {
	case null.String, null.Time, null.Int, null.Float, null.Bool:
		if v, ok := t.(driver.Valuer); ok {
			if val, err := v.Value(); err == nil {
				return val
			}
		}
	}

	return nil
}
