package storage

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type TransactionManager struct{ db *sqlx.DB }

var sqlBuilder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (t *TransactionManager) BeginTx(ctx context.Context) *sqlx.Tx {
	return t.db.MustBeginTx(ctx, nil)
}

func (t *TransactionManager) Rollback(tx *sqlx.Tx) {
	_ = tx.Rollback()
}

func (t *TransactionManager) Commit(tx *sqlx.Tx) error {
	return tx.Commit()
}
