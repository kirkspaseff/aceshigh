package load

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

const narrativeStagingDDL = `
CREATE TEMP TABLE narratives_staging (
	ev_id          VARCHAR(14) NOT NULL,
	aircraft_key   INTEGER NOT NULL,
	narr_accp      TEXT,
	narr_accf      TEXT,
	narr_cause     TEXT,
	narr_inc       TEXT
) ON COMMIT DROP;
`

var narrativeColumns = []string{
	"ev_id", "aircraft_key",
	"narr_accp", "narr_accf", "narr_cause", "narr_inc",
}

const narrativeOrphanCheckSQL = `
SELECT COUNT(*) FROM narratives_staging s
WHERE NOT EXISTS (
	SELECT 1 FROM aircraft a
	WHERE a.ev_id = s.ev_id AND a.aircraft_key = s.aircraft_key
);
`

const narrativeUpsertSQL = `
INSERT INTO narratives (
	ev_id, aircraft_key, narr_accp, narr_accf, narr_cause, narr_inc
)
SELECT
	ev_id, aircraft_key, narr_accp, narr_accf, narr_cause, narr_inc
FROM narratives_staging
ON CONFLICT (ev_id, aircraft_key) DO UPDATE SET
	narr_accp  = EXCLUDED.narr_accp,
	narr_accf  = EXCLUDED.narr_accf,
	narr_cause = EXCLUDED.narr_cause,
	narr_inc   = EXCLUDED.narr_inc;
`

// Narratives bulk-loads narrative rows. FK pre-flight checks against
// the aircraft table — both events and aircraft must be loaded first.
func Narratives(ctx context.Context, pool *pgxpool.Pool, narratives []*domain.Narrative, s *stats.Counter) error {
	if len(narratives) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, narrativeStagingDDL); err != nil {
		return fmt.Errorf("create staging table: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"narratives_staging"},
		narrativeColumns,
		pgx.CopyFromSlice(len(narratives), func(i int) ([]any, error) {
			n := narratives[i]
			return []any{
				n.EvID, n.AircraftKey,
				n.NarrAccp, n.NarrAccf, n.NarrCause, n.NarrInc,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy to staging: %w", err)
	}
	s.Add("narratives.staged", int(rowsCopied))

	var orphans int
	if err := tx.QueryRow(ctx, narrativeOrphanCheckSQL).Scan(&orphans); err != nil {
		return fmt.Errorf("orphan check: %w", err)
	}
	if orphans > 0 {
		s.Add("narratives.orphans", orphans)
		return fmt.Errorf("%d narrative rows reference unknown (ev_id, aircraft_key); load aircraft first", orphans)
	}

	tag, err := tx.Exec(ctx, narrativeUpsertSQL)
	if err != nil {
		return fmt.Errorf("upsert from staging: %w", err)
	}
	s.Add("narratives.upserted", int(tag.RowsAffected()))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
