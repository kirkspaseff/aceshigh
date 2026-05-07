package load

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

const eventSequenceStagingDDL = `
CREATE TEMP TABLE events_sequence_staging (
	ev_id                   VARCHAR(14) NOT NULL,
	aircraft_key            INTEGER NOT NULL,
	occurrence_no           INTEGER NOT NULL,
	occurrence_code         VARCHAR(7),
	occurrence_description  TEXT,
	phase_no                VARCHAR(3),
	eventsoe_no             VARCHAR(3),
	defining_ev             BOOLEAN NOT NULL
) ON COMMIT DROP;
`

var eventSequenceColumns = []string{
	"ev_id", "aircraft_key", "occurrence_no",
	"occurrence_code", "occurrence_description",
	"phase_no", "eventsoe_no",
	"defining_ev",
}

const eventSequenceOrphanCheckSQL = `
SELECT COUNT(*) FROM events_sequence_staging s
WHERE NOT EXISTS (
	SELECT 1 FROM aircraft a
	WHERE a.ev_id = s.ev_id AND a.aircraft_key = s.aircraft_key
);
`

const eventSequenceUpsertSQL = `
INSERT INTO events_sequence (
	ev_id, aircraft_key, occurrence_no,
	occurrence_code, occurrence_description,
	phase_no, eventsoe_no, defining_ev
)
SELECT
	ev_id, aircraft_key, occurrence_no,
	occurrence_code, occurrence_description,
	phase_no, eventsoe_no, defining_ev
FROM events_sequence_staging
ON CONFLICT (ev_id, aircraft_key, occurrence_no) DO UPDATE SET
	occurrence_code        = EXCLUDED.occurrence_code,
	occurrence_description = EXCLUDED.occurrence_description,
	phase_no               = EXCLUDED.phase_no,
	eventsoe_no            = EXCLUDED.eventsoe_no,
	defining_ev            = EXCLUDED.defining_ev;
`

// EventsSequence bulk-loads event sequence rows. FK pre-flight checks
// against aircraft.
func EventsSequence(ctx context.Context, pool *pgxpool.Pool, sequences []*domain.EventSequence, s *stats.Counter) error {
	if len(sequences) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, eventSequenceStagingDDL); err != nil {
		return fmt.Errorf("create staging table: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"events_sequence_staging"},
		eventSequenceColumns,
		pgx.CopyFromSlice(len(sequences), func(i int) ([]any, error) {
			e := sequences[i]
			return []any{
				e.EvID, e.AircraftKey, e.OccurrenceNo,
				e.OccurrenceCode, e.OccurrenceDescription,
				e.PhaseNo, e.EventsoeNo,
				e.DefiningEv,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy to staging: %w", err)
	}
	s.Add("events_sequence.staged", int(rowsCopied))

	var orphans int
	if err := tx.QueryRow(ctx, eventSequenceOrphanCheckSQL).Scan(&orphans); err != nil {
		return fmt.Errorf("orphan check: %w", err)
	}
	if orphans > 0 {
		s.Add("events_sequence.orphans", orphans)
		return fmt.Errorf("%d events_sequence rows reference unknown (ev_id, aircraft_key); load aircraft first", orphans)
	}

	tag, err := tx.Exec(ctx, eventSequenceUpsertSQL)
	if err != nil {
		return fmt.Errorf("upsert from staging: %w", err)
	}
	s.Add("events_sequence.upserted", int(tag.RowsAffected()))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
