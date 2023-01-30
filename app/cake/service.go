package cake

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type (
	Service interface {
		Get(context.Context, GetCmd) (*Cake, error)
		Save(context.Context, *Cake) error
	}

	ServiceImpl struct{}

	GetCmd struct {
		ID, IdentityID uuid.UUID
	}
)

var _ Service = (*ServiceImpl)(nil)

func NewService() *ServiceImpl { return &ServiceImpl{} }

func (s *ServiceImpl) Get(ctx context.Context, cmd GetCmd) (*Cake, error) {
	return New(cmd.IdentityID, "Käsekuchen"), nil
}

func (s *ServiceImpl) Save(ctx context.Context, cake *Cake) error {
	if err := cake.Validate(); err != nil {
		return fmt.Errorf("failed to validate cake: %w", err)
	}

	// Persist to DB

	return nil
}
