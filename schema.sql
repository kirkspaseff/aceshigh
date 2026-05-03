-- Lookup tables first — load these once, reference from everything else
CREATE TABLE code_lookups (
    ct_name     TEXT NOT NULL,        -- e.g. 'phase_flt_spec'
    code        TEXT NOT NULL,        -- e.g. '540'
    meaning     TEXT NOT NULL,        -- e.g. 'Maneuvering'
    PRIMARY KEY (ct_name, code)
);

CREATE TABLE countries (
    country_code  VARCHAR(3) PRIMARY KEY,
    country_name  TEXT NOT NULL
);

CREATE TABLE us_states (
    state         VARCHAR(2) PRIMARY KEY,
    name          TEXT NOT NULL,
    faa_region    VARCHAR(2) NOT NULL
);

-- Core: one row per NTSB investigation
CREATE TABLE events (
    ev_id          VARCHAR(14) PRIMARY KEY,
    ntsb_no        VARCHAR(10),                      -- human-readable, e.g. CEN26FA115
    ev_type        VARCHAR(3),                       -- ACC | INC
    ev_date        DATE,
    ev_time        INTEGER,                          -- HHMM in UTC, e.g. 1907 = 19:07 UTC, 530 = 05:30 UTC
    ev_tmzn        VARCHAR(3),
    
    ev_city        TEXT,
    ev_state       VARCHAR(2),
    ev_country     VARCHAR(4),
    ev_site_zipcode VARCHAR(10),
    coords         GEOGRAPHY(POINT, 4326),           -- built from dec_lat/dec_lon at ingest
    
    light_cond     VARCHAR(4),
    wx_cond_basic  VARCHAR(3),                       -- VMC | IMC | UNK
    
    ev_highest_injury VARCHAR(4),                    -- FATL | SERS | MINR | NONE
    inj_tot_f      INTEGER,
    inj_tot_s      INTEGER,
    inj_tot_m      INTEGER,
    inj_tot_n      INTEGER,
    inj_tot_t      INTEGER,
    
    mid_air        BOOLEAN,                          -- normalize 'Y'/'N'/null to bool/null
    on_ground_collision BOOLEAN,
    
    raw            JSONB NOT NULL,
    ingested_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_lchg_date TIMESTAMPTZ                     -- from upstream lchg_date, for change detection
);

CREATE INDEX events_date_idx ON events (ev_date DESC) WHERE ev_date IS NOT NULL;
CREATE INDEX events_coords_idx ON events USING GIST (coords);
CREATE INDEX events_state_idx ON events (ev_state);
CREATE INDEX events_country_idx ON events (ev_country);
CREATE INDEX events_severity_idx ON events (ev_highest_injury);

CREATE TABLE aircraft (
    ev_id          VARCHAR(14) NOT NULL REFERENCES events(ev_id) ON DELETE CASCADE,
    aircraft_key   INTEGER NOT NULL,
    
    regis_no       VARCHAR(11),                      -- e.g. N530NA, C-GBHZ; trim
    ntsb_no_acft   VARCHAR(11),                      -- per-aircraft suffix variant
    
    acft_make      TEXT,                             -- raw, trim only
    acft_model     TEXT,                             -- raw, trim only
    acft_series    VARCHAR(10),
    acft_year      INTEGER,
    acft_category  VARCHAR(4),                       -- AIR | HELI | GLDR | ...
    homebuilt      BOOLEAN,                          -- normalize Y/N
    unmanned       BOOLEAN NOT NULL DEFAULT FALSE,
    
    far_part       VARCHAR(4),                       -- 091 | 121 | 135 | NUSC | NUSN
    type_fly       VARCHAR(4),                       -- PERS | INST | BUS | ...
    
    damage         VARCHAR(4),                       -- DEST | SUBS | MINR | NONE | null
    acft_fire      VARCHAR(4),
    acft_expl      VARCHAR(4),
    
    total_seats    INTEGER,
    num_eng        SMALLINT,
    
    oper_name      TEXT,                             -- trim
    oper_country   VARCHAR(3),                       -- trim trailing space
    
    dprt_apt_id    VARCHAR(4),                       -- trim
    dprt_city      TEXT,
    dprt_state     VARCHAR(2),
    dprt_country   VARCHAR(3),                       -- trim
    dprt_time      INTEGER,
    
    dest_apt_id    VARCHAR(4),                       -- trim
    dest_city      TEXT,
    dest_state     VARCHAR(2),
    dest_country   VARCHAR(3),                       -- trim
    
    phase_flt_spec INTEGER,                          -- joins to ct_iaids
    
    raw            JSONB NOT NULL,
    PRIMARY KEY (ev_id, aircraft_key)
);

CREATE INDEX aircraft_make_idx ON aircraft (acft_make);
CREATE INDEX aircraft_far_part_idx ON aircraft (far_part);
CREATE INDEX aircraft_type_fly_idx ON aircraft (type_fly);
CREATE INDEX aircraft_damage_idx ON aircraft (damage);

-- Narratives — separate table because the text is large
CREATE TABLE narratives (
    ev_id          VARCHAR(14) NOT NULL,
    aircraft_key   INTEGER NOT NULL,
    narr_accp      TEXT,                             -- accident analysis
    narr_accf      TEXT,                             -- factual
    narr_cause     TEXT,                             -- probable cause
    narr_inc       TEXT,                             -- incident narrative
    PRIMARY KEY (ev_id, aircraft_key),
    FOREIGN KEY (ev_id, aircraft_key) REFERENCES aircraft(ev_id, aircraft_key) ON DELETE CASCADE
);

-- Add full-text search later when you wire that feature up:
-- ALTER TABLE narratives ADD COLUMN narr_tsv tsvector
--   GENERATED ALWAYS AS (to_tsvector('english',
--     coalesce(narr_cause,'') || ' ' || coalesce(narr_accf,'') || ' ' || coalesce(narr_accp,''))) STORED;
-- CREATE INDEX narratives_tsv_idx ON narratives USING GIN (narr_tsv);

-- Findings — post-2008 causal analysis
CREATE TABLE findings (
    ev_id            VARCHAR(14) NOT NULL,
    aircraft_key     INTEGER NOT NULL,
    finding_no       INTEGER NOT NULL,
    finding_code     VARCHAR(10),
    finding_description TEXT,
    cause_factor     VARCHAR(1),                     -- C=cause, F=factor, etc.
    PRIMARY KEY (ev_id, aircraft_key, finding_no),
    FOREIGN KEY (ev_id, aircraft_key) REFERENCES aircraft(ev_id, aircraft_key) ON DELETE CASCADE
);

CREATE INDEX findings_cause_idx ON findings (cause_factor);

-- Sequence of events — what happened, in order
CREATE TABLE events_sequence (
    ev_id          VARCHAR(14) NOT NULL,
    aircraft_key   INTEGER NOT NULL,
    occurrence_no  INTEGER NOT NULL,
    occurrence_code VARCHAR(7),
    occurrence_description TEXT,
    phase_no       VARCHAR(3),
    eventsoe_no    VARCHAR(3),
    defining_ev    BOOLEAN NOT NULL,
    PRIMARY KEY (ev_id, aircraft_key, occurrence_no),
    FOREIGN KEY (ev_id, aircraft_key) REFERENCES aircraft(ev_id, aircraft_key) ON DELETE CASCADE
);

-- Skip flight_crew, engines, injury, occurrences for v1
-- They can be added in subsequent migrations when the UI needs them
