-- +goose Up
-- +goose StatementBegin

-- Country codes from the NTSB country lookup table.
CREATE TABLE countries (
    country_code  VARCHAR(3) PRIMARY KEY,
    country_name  TEXT NOT NULL
);

-- US states with FAA region. Mostly useful for grouping events by region.
CREATE TABLE us_states (
    state         VARCHAR(2) PRIMARY KEY,
    name          TEXT NOT NULL,
    faa_region    VARCHAR(2) NOT NULL
);

-- Generic code-to-meaning lookup loaded from ct_iaids.
-- Composite PK on (ct_name, code) since the same code value can mean
-- different things depending on which column it applies to.
CREATE TABLE code_lookups (
    ct_name       TEXT NOT NULL,
    code          TEXT NOT NULL,
    meaning       TEXT NOT NULL,
    PRIMARY KEY (ct_name, code)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS code_lookups;
DROP TABLE IF EXISTS us_states;
DROP TABLE IF EXISTS countries;
-- +goose StatementEnd
