package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type TransactionManager struct{ db *sqlx.DB }

//nolint:deadcode,varcheck //begone
var sqlBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (t *TransactionManager) MustBegin(ctx context.Context) *sqlx.Tx {
	return t.db.MustBeginTx(ctx, nil)
}
