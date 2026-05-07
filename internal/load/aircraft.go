package load

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

const aircraftStagingDDL = `
CREATE TEMP TABLE aircraft_staging (
	ev_id           VARCHAR(14) NOT NULL,
	aircraft_key    INTEGER NOT NULL,
	regis_no        VARCHAR(11),
	ntsb_no_acft    VARCHAR(11),
	acft_make       TEXT,
	acft_model      TEXT,
	acft_series     VARCHAR(10),
	acft_year       INTEGER,
	acft_category   VARCHAR(4),
	homebuilt       BOOLEAN,
	unmanned        BOOLEAN NOT NULL,
	far_part        VARCHAR(4),
	type_fly        VARCHAR(4),
	damage          VARCHAR(4),
	acft_fire       VARCHAR(4),
	acft_expl       VARCHAR(4),
	total_seats     INTEGER,
	num_eng         SMALLINT,
	oper_name       TEXT,
	oper_country    VARCHAR(3),
	dprt_apt_id     VARCHAR(4),
	dprt_city       TEXT,
	dprt_state      VARCHAR(2),
	dprt_country    VARCHAR(3),
	dprt_time       INTEGER,
	dest_apt_id     VARCHAR(4),
	dest_city       TEXT,
	dest_state      VARCHAR(2),
	dest_country    VARCHAR(3),
	phase_flt_spec  INTEGER,
	raw             JSONB NOT NULL
) ON COMMIT DROP;
`

var aircraftColumns = []string{
	"ev_id", "aircraft_key",
	"regis_no", "ntsb_no_acft",
	"acft_make", "acft_model", "acft_series", "acft_year", "acft_category",
	"homebuilt", "unmanned",
	"far_part", "type_fly",
	"damage", "acft_fire", "acft_expl",
	"total_seats", "num_eng",
	"oper_name", "oper_country",
	"dprt_apt_id", "dprt_city", "dprt_state", "dprt_country", "dprt_time",
	"dest_apt_id", "dest_city", "dest_state", "dest_country",
	"phase_flt_spec",
	"raw",
}

// orphanCheckSQL counts staged aircraft whose ev_id has no matching event.
// We do this BEFORE the upsert so we can give a clean error message
// instead of relying on Postgres's FK violation message.
const aircraftOrphanCheckSQL = `
SELECT COUNT(*) FROM aircraft_staging s
WHERE NOT EXISTS (SELECT 1 FROM events e WHERE e.ev_id = s.ev_id);
`

const aircraftUpsertSQL = `
INSERT INTO aircraft (
	ev_id, aircraft_key,
	regis_no, ntsb_no_acft,
	acft_make, acft_model, acft_series, acft_year, acft_category,
	homebuilt, unmanned,
	far_part, type_fly,
	damage, acft_fire, acft_expl,
	total_seats, num_eng,
	oper_name, oper_country,
	dprt_apt_id, dprt_city, dprt_state, dprt_country, dprt_time,
	dest_apt_id, dest_city, dest_state, dest_country,
	phase_flt_spec,
	raw
)
SELECT
	ev_id, aircraft_key,
	regis_no, ntsb_no_acft,
	acft_make, acft_model, acft_series, acft_year, acft_category,
	homebuilt, unmanned,
	far_part, type_fly,
	damage, acft_fire, acft_expl,
	total_seats, num_eng,
	oper_name, oper_country,
	dprt_apt_id, dprt_city, dprt_state, dprt_country, dprt_time,
	dest_apt_id, dest_city, dest_state, dest_country,
	phase_flt_spec,
	raw
FROM aircraft_staging
ON CONFLICT (ev_id, aircraft_key) DO UPDATE SET
	regis_no       = EXCLUDED.regis_no,
	ntsb_no_acft   = EXCLUDED.ntsb_no_acft,
	acft_make      = EXCLUDED.acft_make,
	acft_model     = EXCLUDED.acft_model,
	acft_series    = EXCLUDED.acft_series,
	acft_year      = EXCLUDED.acft_year,
	acft_category  = EXCLUDED.acft_category,
	homebuilt      = EXCLUDED.homebuilt,
	unmanned       = EXCLUDED.unmanned,
	far_part       = EXCLUDED.far_part,
	type_fly       = EXCLUDED.type_fly,
	damage         = EXCLUDED.damage,
	acft_fire      = EXCLUDED.acft_fire,
	acft_expl      = EXCLUDED.acft_expl,
	total_seats    = EXCLUDED.total_seats,
	num_eng        = EXCLUDED.num_eng,
	oper_name      = EXCLUDED.oper_name,
	oper_country   = EXCLUDED.oper_country,
	dprt_apt_id    = EXCLUDED.dprt_apt_id,
	dprt_city      = EXCLUDED.dprt_city,
	dprt_state     = EXCLUDED.dprt_state,
	dprt_country   = EXCLUDED.dprt_country,
	dprt_time      = EXCLUDED.dprt_time,
	dest_apt_id    = EXCLUDED.dest_apt_id,
	dest_city      = EXCLUDED.dest_city,
	dest_state     = EXCLUDED.dest_state,
	dest_country   = EXCLUDED.dest_country,
	phase_flt_spec = EXCLUDED.phase_flt_spec,
	raw            = EXCLUDED.raw;
`

// Aircraft bulk-loads aircraft rows. Performs an FK pre-flight check
// against events and aborts with a clear error if orphans are present —
// the events table must be loaded first.
func Aircraft(ctx context.Context, pool *pgxpool.Pool, aircraft []*domain.Aircraft, s *stats.Counter) error {
	if len(aircraft) == 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, aircraftStagingDDL); err != nil {
		return fmt.Errorf("create staging table: %w", err)
	}

	rowsCopied, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"aircraft_staging"},
		aircraftColumns,
		pgx.CopyFromSlice(len(aircraft), func(i int) ([]any, error) {
			a := aircraft[i]
			return []any{
				a.EvID, a.AircraftKey,
				a.RegisNo, a.NtsbNoAcft,
				a.AcftMake, a.AcftModel, a.AcftSeries, a.AcftYear, a.AcftCategory,
				a.Homebuilt, a.Unmanned,
				a.FarPart, a.TypeFly,
				a.Damage, a.AcftFire, a.AcftExpl,
				a.TotalSeats, a.NumEng,
				a.OperName, a.OperCountry,
				a.DprtAptID, a.DprtCity, a.DprtState, a.DprtCountry, a.DprtTime,
				a.DestAptID, a.DestCity, a.DestState, a.DestCountry,
				a.PhaseFltSpec,
				string(a.Raw),
			}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("copy to staging: %w", err)
	}
	s.Add("aircraft.staged", int(rowsCopied))

	// Pre-flight FK check: surface orphan rows clearly rather than
	// letting the upsert fail with a less-readable FK violation.
	var orphans int
	if err := tx.QueryRow(ctx, aircraftOrphanCheckSQL).Scan(&orphans); err != nil {
		return fmt.Errorf("orphan check: %w", err)
	}
	if orphans > 0 {
		s.Add("aircraft.orphans", orphans)
		return fmt.Errorf("%d aircraft rows reference unknown ev_id; load events first", orphans)
	}

	tag, err := tx.Exec(ctx, aircraftUpsertSQL)
	if err != nil {
		return fmt.Errorf("upsert from staging: %w", err)
	}
	s.Add("aircraft.upserted", int(tag.RowsAffected()))

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
