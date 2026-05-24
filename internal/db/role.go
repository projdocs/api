package db

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Executable interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func SetUser(e Executable, role string, id uuid.UUID) error {
	statements := []string{
		fmt.Sprintf(`SET LOCAL role = '%s'`, role),
		fmt.Sprintf(`SELECT set_config('request.jwt.claims', '{"role":"%s"}', true)`, role),
		fmt.Sprintf(`SELECT set_config('request.jwt.claim.sub', '%s', true)`, id.String()),
	}

	for _, stmt := range statements {
		if _, err := e.Exec(stmt); err != nil {
			return fmt.Errorf("error setting user (%s): %w", stmt, err)
		}
	}

	return nil
}
