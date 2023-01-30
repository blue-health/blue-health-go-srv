package cake

import (
	"time"

	"github.com/blue-health/blue-health-go-srv/app/util"
	"github.com/google/uuid"
)

type Cake struct {
	ID         uuid.UUID `validate:"required" db:"id" json:"id" yaml:"id"`
	IdentityID uuid.UUID `db:"identity_id" json:"identity_id" yaml:"identityId"`
	Name       string    `db:"name" json:"name" yaml:"name"`
	InsertedAt time.Time `validate:"required" db:"inserted_at" json:"inserted_at" yaml:"insertedAt"`
	UpdatedAt  time.Time `validate:"required" db:"updated_at" json:"updated_at" yaml:"updatedAt"`
}

func New(identityID uuid.UUID, name string) *Cake {
	now := time.Now().UTC()

	return &Cake{
		ID:         uuid.New(),
		IdentityID: identityID,
		Name:       name,
		InsertedAt: now,
		UpdatedAt:  now,
	}
}

func (c *Cake) Validate() error {
	return util.Validate.Struct(c)
}
