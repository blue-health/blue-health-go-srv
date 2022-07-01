package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/blue-health/blue-health-go-srv/app/util"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

type (
	r struct {
		Error m `json:"error"`
	}

	f struct {
		Name       string `json:"name"`
		Validation string `json:"validation"`
		Param      string `json:"param"`
	}

	m struct {
		Message string `json:"message"`
		Fields  []f    `json:"fields"`
	}
)

var mappedErrors = map[error]int{}

func ErrorC(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(r{Error: m{Message: http.StatusText(code)}})
}

func Error(w http.ResponseWriter, err error) {
	var (
		originalErr = err
		code        = http.StatusInternalServerError
		message     string
		fields      []f
	)

	err = unwrap(err)

	switch v := err.(type) {
	case *util.ValidationError:
		code = http.StatusBadRequest
		message = v.Root.Error()

		if c, ok := mappedErrors[v.Root]; ok {
			code = c
		}

		fields = make([]f, 0, len(v.Errors))

		for i := range v.Errors {
			fields = append(fields, f{
				Name:       toSnakeCase(v.Errors[i].Field()),
				Validation: v.Errors[i].ActualTag(),
				Param:      v.Errors[i].Param(),
			})
		}

	case validator.ValidationErrors:
		code = http.StatusBadRequest
		message = http.StatusText(code)
		fields = make([]f, 0, len(v))

		for i := range v {
			fields = append(fields, f{
				Name:       toSnakeCase(v[i].Field()),
				Validation: v[i].ActualTag(),
				Param:      v[i].Param(),
			})
		}

	default:
		message = err.Error()

		if c, ok := mappedErrors[err]; ok {
			code = c
		}
	}

	switch {
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		log.WithFields(log.Fields{"http_code": code, "error_code": message, "err": originalErr}).Warn("application error")
	case code >= http.StatusInternalServerError:
		log.WithFields(log.Fields{"http_code": code, "error_code": message, "err": originalErr}).Warn("server error")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(r{Error: m{Message: message, Fields: fields}})
}

func unwrap(err error) error {
	for errors.Unwrap(err) != nil {
		err = errors.Unwrap(err)
	}

	return err
}

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")

	return strings.ToLower(snake)
}
