-- +goose Up
-- +goose StatementBegin

-- Post-investigation causal analysis. Modern (post-2008) records use this.
-- cause_factor: C = cause, F = factor, etc.
CREATE TABLE findings (
    ev_id                  VARCHAR(14) NOT NULL,
    aircraft_key           INTEGER NOT NULL,
    finding_no             INTEGER NOT NULL,
    finding_code           VARCHAR(10),
    finding_description    TEXT,
    cause_factor           VARCHAR(1),
    PRIMARY KEY (ev_id, aircraft_key, finding_no),
    FOREIGN KEY (ev_id, aircraft_key) REFERENCES aircraft(ev_id, aircraft_key) ON DELETE CASCADE
);

CREATE INDEX findings_cause_idx ON findings (cause_factor);

-- Sequence of what happened during the event. Modern timeline
-- representation; older records use the legacy `occurrences` table
-- which we're not ingesting for v1.
CREATE TABLE events_sequence (
    ev_id                    VARCHAR(14) NOT NULL,
    aircraft_key             INTEGER NOT NULL,
    occurrence_no            INTEGER NOT NULL,
    occurrence_code          VARCHAR(7),
    occurrence_description   TEXT,
    phase_no                 VARCHAR(3),
    eventsoe_no              VARCHAR(3),
    defining_ev              BOOLEAN NOT NULL,
    PRIMARY KEY (ev_id, aircraft_key, occurrence_no),
    FOREIGN KEY (ev_id, aircraft_key) REFERENCES aircraft(ev_id, aircraft_key) ON DELETE CASCADE
);

CREATE INDEX events_sequence_defining_idx ON events_sequence (defining_ev) WHERE defining_ev = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events_sequence;
DROP TABLE IF EXISTS findings;
-- +goose StatementEnd
