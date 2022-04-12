package util

import "github.com/go-playground/validator/v10"

type ValidationError struct {
	Root   error
	Errors validator.ValidationErrors
}

func (e *ValidationError) Error() string {
	return e.Root.Error()
}

func FailValidation(root, err error) error {
	var v ValidationError

	v.Root = root

	if e, ok := err.(validator.ValidationErrors); ok {
		v.Errors = e
	}

	return &v
}
