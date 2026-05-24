package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

func WithRLS(
	ctx context.Context,
	db *sql.DB,
	role string,
	id uuid.UUID,
) (*sql.Tx, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	if err := SetUser(tx, role, id); err != nil {
		tx.Rollback()
		return nil, err
	}

	return tx, nil
}
