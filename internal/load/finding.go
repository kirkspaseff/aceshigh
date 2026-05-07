package load

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

const findingStagingDDL = `
CREATE TEMP TABLE findings_staging (
	ev_id                VARCHAR(14) NOT NULL,
	aircraft_key         INTEGER NOT NULL,
	finding_no           INTEGER NOT NULL,
	finding_code         VARCHAR(10),
	finding_description  TEXT,
	cause_factor         VARCHAR(1)
) ON COMMIT DROP;
`

var findingColumns = []string{
	"ev_id", "aircraft_key", "finding_no",
	"finding_code", "finding_description", "cause_factor",
}

const findingOrphanCheckSQL = `
SELECT COUNT(*) FROM findings_staging s
WHERE NOT EXISTS (
	SELECT 1 FROM aircraft a
	WHERE a.ev_id = s.ev_id AND a.aircraft_key = s.aircraft_key
);
`

const findingUpsertSQL = `
INSERT INTO findings (
	ev_id, aircraft_key, finding_no,
	finding_code, finding_description, cause_factor
)
SELECT
	ev_id, aircraft_key, finding_no,
	finding_code, finding_description, cause_factor
FROM findings_staging
ON CONFLICT (ev_id, aircraft_key, finding_no) DO UPDATE SET
	finding_code        = EXCLUDED.finding_code,
	finding_description = EXCLUDED.finding_description,
	cause_factor        = EXCLUDED.cause_factor;
`

// Findings bulk-loads finding rows. FK pre-flight checks against aircraft.
func Findings(ctx context.Context, pool *pgxpool.Pool, findings []*domain.Finding, s *stats.Counter) error {
	if len(findings) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, findingStagingDDL); err != nil {
		return fmt.Errorf("create staging table: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"findings_staging"},
		findingColumns,
		pgx.CopyFromSlice(len(findings), func(i int) ([]any, error) {
			f := findings[i]
			return []any{
				f.EvID, f.AircraftKey, f.FindingNo,
				f.FindingCode, f.FindingDescription, f.CauseFactor,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy to staging: %w", err)
	}
	s.Add("findings.staged", int(rowsCopied))

	var orphans int
	if err := tx.QueryRow(ctx, findingOrphanCheckSQL).Scan(&orphans); err != nil {
		return fmt.Errorf("orphan check: %w", err)
	}
	if orphans > 0 {
		s.Add("findings.orphans", orphans)
		return fmt.Errorf("%d finding rows reference unknown (ev_id, aircraft_key); load aircraft first", orphans)
	}

	tag, err := tx.Exec(ctx, findingUpsertSQL)
	if err != nil {
		return fmt.Errorf("upsert from staging: %w", err)
	}
	s.Add("findings.upserted", int(tag.RowsAffected()))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
