-- +goose Up
-- +goose StatementBegin

-- Narrative text fields associated with each (event, aircraft) pair.
-- Often sparse for foreign events (NTSB tracks but doesn't investigate).
-- ~84% of rows have substantive narr_cause text.
CREATE TABLE narratives (
    ev_id          VARCHAR(14) NOT NULL,
    aircraft_key   INTEGER NOT NULL,
    narr_accp      TEXT,                     -- accident analysis
    narr_accf      TEXT,                     -- factual narrative
    narr_cause     TEXT,                     -- probable cause
    narr_inc       TEXT,                     -- incident narrative
    PRIMARY KEY (ev_id, aircraft_key),
    FOREIGN KEY (ev_id, aircraft_key) REFERENCES aircraft(ev_id, aircraft_key) ON DELETE CASCADE
);

-- Full-text search index will be added in a later migration when
-- the search feature is wired up. Leaving it out for now keeps writes fast.

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS narratives;
-- +goose StatementEnd
