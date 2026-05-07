package transform

import (
	"fmt"

	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// Country converts one row from the country CSV. Both columns are required.
func Country(r *acsv.Reader, s *stats.Counter) (*domain.Country, error) {
	code := acsv.StringRequired(r.Get("country_code"))
	name := acsv.StringRequired(r.Get("country_name"))
	if code == "" {
		s.Inc("countries.skipped.missing_code")
		return nil, fmt.Errorf("row missing country_code")
	}
	if name == "" {
		s.Inc("countries.skipped.missing_name")
		return nil, fmt.Errorf("row missing country_name (code=%s)", code)
	}
	return &domain.Country{CountryCode: code, CountryName: name}, nil
}

// USState converts one row from the states CSV. The FAA region is
// non-null in the source schema, so we treat it as required.
func USState(r *acsv.Reader, s *stats.Counter) (*domain.USState, error) {
	state := acsv.StringRequired(r.Get("state"))
	name := acsv.StringRequired(r.Get("name"))
	region := acsv.StringRequired(r.Get("faa_region"))
	if state == "" {
		s.Inc("us_states.skipped.missing_state")
		return nil, fmt.Errorf("row missing state")
	}
	if name == "" || region == "" {
		s.Inc("us_states.skipped.incomplete")
		return nil, fmt.Errorf("incomplete row (state=%s)", state)
	}
	return &domain.USState{State: state, Name: name, FAARegion: region}, nil
}

// CodeLookup converts one row from the ct_iaids CSV. We load every row,
// including those marked not_for_ntsb_use=1 — they're useful for
// translating archived codes that may appear in older data.
func CodeLookup(r *acsv.Reader, s *stats.Counter) (*domain.CodeLookup, error) {
	ctName := acsv.StringRequired(r.Get("ct_name"))
	code := acsv.StringRequired(r.Get("code_iaids"))
	meaning := acsv.StringRequired(r.Get("meaning"))
	if ctName == "" || code == "" {
		s.Inc("code_lookups.skipped.missing_key")
		return nil, fmt.Errorf("row missing ct_name or code_iaids")
	}
	if meaning == "" {
		s.Inc("code_lookups.skipped.missing_meaning")
		return nil, fmt.Errorf("row missing meaning (ct_name=%s code=%s)", ctName, code)
	}
	return &domain.CodeLookup{CTName: ctName, Code: code, Meaning: meaning}, nil
}
