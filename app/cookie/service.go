package cookie

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type (
	Service interface {
		Get(context.Context, GetCmd) (*Cookie, error)
		Save(context.Context, *Cookie) error
	}

	ServiceImpl struct{}

	GetCmd struct {
		ID, IdentityID uuid.UUID
	}
)

var _ Service = (*ServiceImpl)(nil)

func NewService() *ServiceImpl { return &ServiceImpl{} }

func (s *ServiceImpl) Get(ctx context.Context, cmd GetCmd) (*Cookie, error) {
	return New(cmd.IdentityID, "Zimtstern"), nil
}

func (s *ServiceImpl) Save(ctx context.Context, cookie *Cookie) error {
	if err := cookie.Validate(); err != nil {
		return fmt.Errorf("failed to validate cookie: %w", err)
	}

	// Persist to DB

	return nil
}
