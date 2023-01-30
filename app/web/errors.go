package web

import (
	"github.com/blue-health/blue-go-toolbox/logger"
)

var (
	mappedErrors = map[error]int{}

	documentedErrors = map[error]string{}
)

func init() {
	logger.RegisterErrors(mappedErrors)
}

func MappedErrors() map[error]int {
	return mappedErrors
}

func DocumentedErrors() map[error]string {
	return documentedErrors
}
