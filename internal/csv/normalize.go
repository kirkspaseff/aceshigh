// Package csv provides helpers for reading and normalizing CSV data
// produced by mdb-export. The conventions here match the patterns seen
// in NTSB exports: trailing whitespace, mixed boolean encodings,
// empty-string-as-null, etc.
package csv

import (
	"strconv"
	"strings"
	"time"
)

// trimAll trims leading and trailing whitespace.
// We apply this to every string field at read time.
func trimAll(s string) string {
	return strings.TrimSpace(s)
}

// String returns a non-nil pointer if the field has content after trimming,
// else nil. This is the right helper for nullable string columns.
func String(s string) *string {
	t := trimAll(s)
	if t == "" {
		return nil
	}
	return &t
}

// StringRequired returns the trimmed string. Use only for fields that must
// always be present (primary keys, etc.).
func StringRequired(s string) string {
	return trimAll(s)
}

// Int parses an int from a string. Empty / whitespace-only / unparseable
// returns nil. We treat parse failures as "missing" rather than erroring,
// because the source has data-entry errors we don't want to halt on.
func Int(s string) *int {
	t := trimAll(s)
	if t == "" {
		return nil
	}
	n, err := strconv.Atoi(t)
	if err != nil {
		return nil
	}
	return &n
}

// Float64 parses a float. Empty / unparseable returns nil.
func Float64(s string) *float64 {
	t := trimAll(s)
	if t == "" {
		return nil
	}
	f, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return nil
	}
	return &f
}

// Bool normalizes the various boolean encodings the NTSB MDB uses:
//   - "Y" / "N" / "" / "U" (most string fields)
//   - "1" / "0" (Postgres-bool exports of NOT NULL columns)
//   - "true" / "false" (defensive)
//
// Unknown / empty returns nil. "U" (unknown) is treated as nil.
func Bool(s string) *bool {
	t := strings.ToUpper(trimAll(s))
	switch t {
	case "Y", "YES", "TRUE", "T", "1":
		v := true
		return &v
	case "N", "NO", "FALSE", "F", "0":
		v := false
		return &v
	}
	return nil
}

// Date parses a "YYYY-MM-DD HH:MM:SS" datetime (the format mdb-export
// produces with -T '%Y-%m-%d %H:%M:%S') into a time.Time pointer.
// Empty / unparseable returns nil.
//
// We deliberately keep the time component even though ev_date is always
// midnight in source data — callers can truncate to a date if they want.
func Date(s string) *time.Time {
	t := trimAll(s)
	if t == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", t)
	if err != nil {
		return nil
	}
	return &parsed
}

// CoordPair returns lat/lon pointers only if both parse and pass range
// validation. If either is missing or out-of-range, both come back nil
// (we don't half-populate a coord pair).
//
// Returns (lat, lon, ok). ok==false signals the caller may want to log
// a normalization counter.
func CoordPair(latStr, lonStr string) (*float64, *float64, bool) {
	lat := Float64(latStr)
	lon := Float64(lonStr)
	if lat == nil || lon == nil {
		return nil, nil, false
	}
	if *lat < -90 || *lat > 90 || *lon < -180 || *lon > 180 {
		return nil, nil, false
	}
	return lat, lon, true
}

