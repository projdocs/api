package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func WithRLS(ctx context.Context, db *sql.DB, id uuid.UUID) (*sql.Tx, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	statements := []string{
		fmt.Sprintf(`SET LOCAL role = '%s'`, "authenticated"),
		fmt.Sprintf(`SELECT set_config('request.jwt.claims', '{"role":"%s"}', true)`, "authenticated"),
		fmt.Sprintf(`SELECT set_config('request.jwt.claim.sub', '%s', true)`, id.String()),
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("setting supabase auth context: %w", err)
		}
	}

	return tx, nil
}
