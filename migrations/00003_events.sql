-- +goose Up
-- +goose StatementBegin

-- One row per NTSB investigation. Parent of aircraft, narratives, etc.
-- ev_id is the internal NTSB identifier (e.g. 20080211X00175).
-- ntsb_no is the human-readable accident number (e.g. DFW08RA039).
CREATE TABLE events (
    ev_id              VARCHAR(14) PRIMARY KEY,
    ntsb_no            VARCHAR(10),
    ev_type            VARCHAR(3),                     -- ACC | INC
    ev_date            DATE,
    ev_time            INTEGER,                        -- HHMM in UTC, e.g. 1907 = 19:07
    ev_tmzn            VARCHAR(3),                     -- always 'UTC' in modern records

    ev_city            TEXT,
    ev_state           VARCHAR(2),                     -- US states only; null otherwise
    ev_country         VARCHAR(3),
    ev_site_zipcode    VARCHAR(10),
    coords             GEOGRAPHY(POINT, 4326),         -- built from dec_lat/dec_lon

    light_cond         VARCHAR(4),                     -- DAYL | NITE | DUSK | DAWN | ...
    wx_cond_basic      VARCHAR(3),                     -- VMC | IMC | UNK

    ev_highest_injury  VARCHAR(4),                     -- FATL | SERS | MINR | NONE
    inj_tot_f          INTEGER,
    inj_tot_s          INTEGER,
    inj_tot_m          INTEGER,
    inj_tot_n          INTEGER,
    inj_tot_t          INTEGER,

    mid_air                BOOLEAN,
    on_ground_collision    BOOLEAN,

    -- Full original record. Useful for surfacing fields we didn't model
    -- and for debugging ingestion without re-running it.
    raw                JSONB NOT NULL,

    ingested_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Upstream lchg_date — used for incremental refresh diffing.
    source_lchg_date   TIMESTAMPTZ
);

CREATE INDEX events_date_idx       ON events (ev_date DESC) WHERE ev_date IS NOT NULL;
CREATE INDEX events_coords_idx     ON events USING GIST (coords);
CREATE INDEX events_state_idx      ON events (ev_state);
CREATE INDEX events_country_idx    ON events (ev_country);
CREATE INDEX events_severity_idx   ON events (ev_highest_injury);
CREATE INDEX events_lchg_idx       ON events (source_lchg_date);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
