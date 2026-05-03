-- +goose Up
-- +goose StatementBegin

-- One row per aircraft per event. Multi-aircraft events (e.g. midairs)
-- have multiple rows sharing the same ev_id with distinct aircraft_key.
CREATE TABLE aircraft (
    ev_id                    VARCHAR(14) NOT NULL REFERENCES events(ev_id) ON DELETE CASCADE,
    aircraft_key             INTEGER NOT NULL,

    regis_no                 VARCHAR(11),                -- tail number, e.g. N530NA, C-GBHZ
    ntsb_no_acft             VARCHAR(11),                -- per-aircraft accident number suffix

    acft_make                TEXT,                       -- raw, dirty: PIPER vs Piper etc.
    acft_model               TEXT,
    acft_series              VARCHAR(10),
    acft_year                INTEGER,
    acft_category            VARCHAR(4),                 -- AIR | HELI | GLDR | ...
    homebuilt                BOOLEAN,
    unmanned                 BOOLEAN NOT NULL DEFAULT FALSE,

    far_part                 VARCHAR(4),                 -- 091 | 121 | 135 | NUSC | NUSN
    type_fly                 VARCHAR(4),                 -- PERS | INST | BUS | ...

    damage                   VARCHAR(4),                 -- DEST | SUBS | MINR | NONE
    acft_fire                VARCHAR(4),
    acft_expl                VARCHAR(4),

    total_seats              INTEGER,
    num_eng                  SMALLINT,

    oper_name                TEXT,
    oper_country             VARCHAR(3),

    dprt_apt_id              VARCHAR(4),
    dprt_city                TEXT,
    dprt_state               VARCHAR(2),
    dprt_country             VARCHAR(3),
    dprt_time                INTEGER,                    -- HHMM

    dest_apt_id              VARCHAR(4),
    dest_city                TEXT,
    dest_state               VARCHAR(2),
    dest_country             VARCHAR(3),

    phase_flt_spec           INTEGER,                    -- joins to code_lookups

    raw                      JSONB NOT NULL,

    PRIMARY KEY (ev_id, aircraft_key)
);

CREATE INDEX aircraft_make_idx     ON aircraft (acft_make);
CREATE INDEX aircraft_far_part_idx ON aircraft (far_part);
CREATE INDEX aircraft_type_fly_idx ON aircraft (type_fly);
CREATE INDEX aircraft_damage_idx   ON aircraft (damage);
CREATE INDEX aircraft_category_idx ON aircraft (acft_category);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS aircraft;
-- +goose StatementEnd
