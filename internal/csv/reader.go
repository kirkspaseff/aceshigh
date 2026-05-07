package csv

import (
	stdcsv "encoding/csv"
	"fmt"
	"io"
	"strings"
)

// Reader wraps the standard csv.Reader to provide name-based field access.
// mdb-export output has header row + many data rows; we want to address
// fields by column name rather than positional index because (a) it's
// readable and (b) column ordering can shift between exports.
//
// Column lookups are case-insensitive: the underlying MDB table case
// varies (events.csv has "ev_id", aircraft.csv has "Aircraft_Key" with
// title case, etc.) and forcing callers to remember which is which is
// a recipe for off-by-one debug sessions.
type Reader struct {
	r       *stdcsv.Reader
	headers map[string]int // keyed by lowercased column name
	row     []string
}

// NewReader creates a new column-name-aware CSV reader from r.
// It consumes the header row immediately. Returns an error if the file
// is empty or unreadable.
func NewReader(r io.Reader) (*Reader, error) {
	c := stdcsv.NewReader(r)
	// The mdb-export output sometimes has rows with fewer fields than
	// the header (trailing empty values trimmed). FieldsPerRecord = -1
	// disables the consistency check so we don't error on those.
	c.FieldsPerRecord = -1
	// LazyQuotes = false (the default) is correct for our well-quoted
	// input — strict RFC 4180 parsing is what we want.

	header, err := c.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	headers := make(map[string]int, len(header))
	for i, name := range header {
		headers[strings.ToLower(name)] = i
	}
	return &Reader{r: c, headers: headers}, nil
}

// Read advances to the next data row. Returns io.EOF when done.
// The row is held internally; access fields via Get / GetRaw.
func (r *Reader) Read() error {
	row, err := r.r.Read()
	if err != nil {
		return err
	}
	r.row = row
	return nil
}

// Get returns the value of column `name` in the current row.
// Lookup is case-insensitive. Returns "" if the column doesn't exist
// or the row has fewer fields than the header — both treated as
// missing-data, not errors.
func (r *Reader) Get(name string) string {
	idx, ok := r.headers[strings.ToLower(name)]
	if !ok {
		return ""
	}
	if idx >= len(r.row) {
		return ""
	}
	return r.row[idx]
}

// Row returns a map of the current row's fields, keyed by lowercased
// column name. Useful for building the raw JSONB blob.
func (r *Reader) Row() map[string]string {
	out := make(map[string]string, len(r.headers))
	for name, idx := range r.headers {
		if idx < len(r.row) {
			out[name] = r.row[idx]
		}
	}
	return out
}

// Headers returns the lowercased column names in the order they appeared.
// (Unused for now; useful later for diagnostics.)
func (r *Reader) Headers() []string {
	out := make([]string, len(r.headers))
	for name, idx := range r.headers {
		out[idx] = name
	}
	return out
}

