// Lookup table loaders use truncate-and-replace: small reference data
// that doesn't change often, so simplest-thing-that-works wins. Both
// the truncate and the COPY happen in one transaction — if the load
// fails partway, the table doesn't end up empty.

package load

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// Countries replaces all rows in the countries table.
func Countries(ctx context.Context, pool *pgxpool.Pool, countries []*domain.Country, s *stats.Counter) error {
	if len(countries) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `TRUNCATE TABLE countries;`); err != nil {
		return fmt.Errorf("truncate countries: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"countries"},
		[]string{"country_code", "country_name"},
		pgx.CopyFromSlice(len(countries), func(i int) ([]any, error) {
			c := countries[i]
			return []any{c.CountryCode, c.CountryName}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy countries: %w", err)
	}
	s.Add("countries.loaded", int(rowsCopied))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// USStates replaces all rows in the us_states table.
func USStates(ctx context.Context, pool *pgxpool.Pool, states []*domain.USState, s *stats.Counter) error {
	if len(states) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `TRUNCATE TABLE us_states;`); err != nil {
		return fmt.Errorf("truncate us_states: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"us_states"},
		[]string{"state", "name", "faa_region"},
		pgx.CopyFromSlice(len(states), func(i int) ([]any, error) {
			st := states[i]
			return []any{st.State, st.Name, st.FAARegion}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy us_states: %w", err)
	}
	s.Add("us_states.loaded", int(rowsCopied))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// CodeLookups replaces all rows in the code_lookups table.
func CodeLookups(ctx context.Context, pool *pgxpool.Pool, lookups []*domain.CodeLookup, s *stats.Counter) error {
	if len(lookups) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `TRUNCATE TABLE code_lookups;`); err != nil {
		return fmt.Errorf("truncate code_lookups: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"code_lookups"},
		[]string{"ct_name", "code", "meaning"},
		pgx.CopyFromSlice(len(lookups), func(i int) ([]any, error) {
			l := lookups[i]
			return []any{l.CTName, l.Code, l.Meaning}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy code_lookups: %w", err)
	}
	s.Add("code_lookups.loaded", int(rowsCopied))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
