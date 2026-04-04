package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type Base struct {
	db *sql.DB
}

func NewBase(db *sql.DB) Base {
	return Base{db: db}
}

func toJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func fromJSON[T any](raw []byte, out *T) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func mapSQLError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrValidation
		}
	}
	return err
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

func queryRowContext(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	return db.QueryRowContext(ctx, query, args...)
}
