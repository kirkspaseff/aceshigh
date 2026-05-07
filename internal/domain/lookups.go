package domain

// Country is a row from the NTSB country lookup table.
type Country struct {
	CountryCode string
	CountryName string
}

// USState is a row from the NTSB states lookup table.
// USState (rather than State) avoids confusion with the
// "state" pattern in domain modeling.
type USState struct {
	State     string // 2-letter abbreviation
	Name      string // full state name
	FAARegion string // 2-letter FAA region code
}

// CodeLookup is one entry in the generic ct_iaids lookup table.
// (ct_name, code) is the natural key — the same code value can mean
// different things depending on which column it applies to.
type CodeLookup struct {
	CTName  string // e.g. "phase_flt_spec"
	Code    string // e.g. "540"
	Meaning string // e.g. "Maneuvering"
}
