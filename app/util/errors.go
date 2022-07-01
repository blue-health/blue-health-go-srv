package util

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidationError struct {
	Root      error
	Namespace string
	Errors    validator.ValidationErrors
}

func (e *ValidationError) Error() string {
	var sb strings.Builder

	if e.Root != nil {
		sb.WriteString(e.Root.Error())
	}

	if e.Namespace != "" {
		sb.WriteString(" (")
		sb.WriteString(e.Namespace)
		sb.WriteString(")")
	}

	if e.Errors != nil && len(e.Errors) > 0 {
		sb.WriteString(": ")
		sb.WriteString(e.Errors.Error())
	}

	return sb.String()
}

func FailValidation(root, err error) error {
	var v ValidationError

	v.Root = root

	if e, ok := err.(validator.ValidationErrors); ok {
		v.Errors = e
	}

	return &v
}

func FailNamespacedValidation(root, err error, namespace string) error {
	var v ValidationError

	v.Root = root
	v.Namespace = namespace

	if e, ok := err.(validator.ValidationErrors); ok {
		v.Errors = e
	}

	return &v
}
