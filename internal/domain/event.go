// Package domain holds the typed Go representations of the database rows.
// These types are shared between the loader and (eventually) the API.
package domain

import (
	"encoding/json"
	"time"
)

// Event is one NTSB investigation, mapped to the events table.
//
// Fields use pointers when null is meaningful (the source data has many
// genuinely-missing values that should remain NULL in Postgres rather
// than getting coerced to zero values).
type Event struct {
	EvID   string  // primary key, e.g. "20080211X00175"
	NtsbNo *string // human-readable, e.g. "DFW08RA039"
	EvType *string // ACC | INC

	EvDate *time.Time // date only; time-of-day always 00:00:00 in source
	EvTime *int       // HHMM in UTC, e.g. 1907 = 19:07
	EvTmzn *string    // always "UTC" in modern records, but kept for honesty

	EvCity        *string
	EvState       *string // 2-letter US state; null for non-US events
	EvCountry     *string // 3-letter ISO-ish, e.g. "USA", "CAN"
	EvSiteZipcode *string

	// PostGIS coordinates. Built from dec_latitude / dec_longitude.
	// Latitude / Longitude are nil-able as a pair: either both set or both nil.
	Latitude  *float64
	Longitude *float64

	LightCond   *string // DAYL | NITE | DUSK | DAWN
	WxCondBasic *string // VMC | IMC | UNK

	EvHighestInjury *string // FATL | SERS | MINR | NONE
	InjTotF         *int
	InjTotS         *int
	InjTotM         *int
	InjTotN         *int
	InjTotT         *int

	MidAir            *bool
	OnGroundCollision *bool

	// Raw is the original record as JSON. Always non-nil after transform.
	Raw json.RawMessage

	// SourceLchgDate is the upstream lchg_date — used for incremental
	// refresh later. May be nil for very old records.
	SourceLchgDate *time.Time
}

