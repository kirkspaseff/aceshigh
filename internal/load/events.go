// Package load handles writing typed domain rows to Postgres.
//
// Strategy: bulk insert via pgx CopyFrom into a temporary staging table
// (one txn per table), then INSERT...ON CONFLICT from the staging table
// into the real table. This pattern:
//
//   - keeps CopyFrom simple (no PostGIS expression handling)
//   - lets us upsert (CopyFrom doesn't support ON CONFLICT directly)
//   - is fast: ~30k rows lands in well under a second
package load

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// eventStagingDDL creates the per-transaction temp table.
// ON COMMIT DROP means it's gone when the txn ends — no cleanup needed.
const eventStagingDDL = `
CREATE TEMP TABLE events_staging (
	ev_id              VARCHAR(14) NOT NULL,
	ntsb_no            VARCHAR(10),
	ev_type            VARCHAR(3),
	ev_date            DATE,
	ev_time            INTEGER,
	ev_tmzn            VARCHAR(3),
	ev_city            TEXT,
	ev_state           VARCHAR(2),
	ev_country         VARCHAR(3),
	ev_site_zipcode    VARCHAR(10),
	latitude           DOUBLE PRECISION,
	longitude          DOUBLE PRECISION,
	light_cond         VARCHAR(4),
	wx_cond_basic      VARCHAR(3),
	ev_highest_injury  VARCHAR(4),
	inj_tot_f          INTEGER,
	inj_tot_s          INTEGER,
	inj_tot_m          INTEGER,
	inj_tot_n          INTEGER,
	inj_tot_t          INTEGER,
	mid_air            BOOLEAN,
	on_ground_collision BOOLEAN,
	raw                JSONB NOT NULL,
	source_lchg_date   TIMESTAMPTZ
) ON COMMIT DROP;
`

// eventColumns must match the column order in the CopyFrom call below
// AND the staging DDL above.
var eventColumns = []string{
	"ev_id", "ntsb_no", "ev_type", "ev_date", "ev_time", "ev_tmzn",
	"ev_city", "ev_state", "ev_country", "ev_site_zipcode",
	"latitude", "longitude",
	"light_cond", "wx_cond_basic",
	"ev_highest_injury",
	"inj_tot_f", "inj_tot_s", "inj_tot_m", "inj_tot_n", "inj_tot_t",
	"mid_air", "on_ground_collision",
	"raw", "source_lchg_date",
}

// upsertSQL moves rows from the staging table into events, building the
// PostGIS POINT from latitude / longitude and upserting on ev_id.
const upsertSQL = `
INSERT INTO events (
	ev_id, ntsb_no, ev_type, ev_date, ev_time, ev_tmzn,
	ev_city, ev_state, ev_country, ev_site_zipcode,
	coords,
	light_cond, wx_cond_basic,
	ev_highest_injury,
	inj_tot_f, inj_tot_s, inj_tot_m, inj_tot_n, inj_tot_t,
	mid_air, on_ground_collision,
	raw, source_lchg_date
)
SELECT
	ev_id, ntsb_no, ev_type, ev_date, ev_time, ev_tmzn,
	ev_city, ev_state, ev_country, ev_site_zipcode,
	CASE
		WHEN latitude IS NOT NULL AND longitude IS NOT NULL
		THEN ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography
		ELSE NULL
	END AS coords,
	light_cond, wx_cond_basic,
	ev_highest_injury,
	inj_tot_f, inj_tot_s, inj_tot_m, inj_tot_n, inj_tot_t,
	mid_air, on_ground_collision,
	raw, source_lchg_date
FROM events_staging
ON CONFLICT (ev_id) DO UPDATE SET
	ntsb_no            = EXCLUDED.ntsb_no,
	ev_type            = EXCLUDED.ev_type,
	ev_date            = EXCLUDED.ev_date,
	ev_time            = EXCLUDED.ev_time,
	ev_tmzn            = EXCLUDED.ev_tmzn,
	ev_city            = EXCLUDED.ev_city,
	ev_state           = EXCLUDED.ev_state,
	ev_country         = EXCLUDED.ev_country,
	ev_site_zipcode    = EXCLUDED.ev_site_zipcode,
	coords             = EXCLUDED.coords,
	light_cond         = EXCLUDED.light_cond,
	wx_cond_basic      = EXCLUDED.wx_cond_basic,
	ev_highest_injury  = EXCLUDED.ev_highest_injury,
	inj_tot_f          = EXCLUDED.inj_tot_f,
	inj_tot_s          = EXCLUDED.inj_tot_s,
	inj_tot_m          = EXCLUDED.inj_tot_m,
	inj_tot_n          = EXCLUDED.inj_tot_n,
	inj_tot_t          = EXCLUDED.inj_tot_t,
	mid_air            = EXCLUDED.mid_air,
	on_ground_collision = EXCLUDED.on_ground_collision,
	raw                = EXCLUDED.raw,
	source_lchg_date   = EXCLUDED.source_lchg_date,
	updated_at         = now();
`

// Events bulk-loads the given events into the events table.
// Uses a single transaction: stage via CopyFrom, upsert via INSERT...SELECT.
func Events(ctx context.Context, pool *pgxpool.Pool, events []*domain.Event, s *stats.Counter) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, eventStagingDDL); err != nil {
		return fmt.Errorf("create staging table: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"events_staging"},
		eventColumns,
		pgx.CopyFromSlice(len(events), func(i int) ([]any, error) {
			e := events[i]
			return []any{
				e.EvID,
				e.NtsbNo,
				e.EvType,
				e.EvDate,
				e.EvTime,
				e.EvTmzn,
				e.EvCity,
				e.EvState,
				e.EvCountry,
				e.EvSiteZipcode,
				e.Latitude,
				e.Longitude,
				e.LightCond,
				e.WxCondBasic,
				e.EvHighestInjury,
				e.InjTotF,
				e.InjTotS,
				e.InjTotM,
				e.InjTotN,
				e.InjTotT,
				e.MidAir,
				e.OnGroundCollision,
				string(e.Raw),
				e.SourceLchgDate,
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy to staging: %w", err)
	}
	s.Add("events.staged", int(rowsCopied))

	tag, err := tx.Exec(ctx, upsertSQL)
	if err != nil {
		return fmt.Errorf("upsert from staging: %w", err)
	}
	s.Add("events.upserted", int(tag.RowsAffected()))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

