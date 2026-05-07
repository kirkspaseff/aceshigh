package domain

// Finding is the post-investigation causal analysis for an (event, aircraft)
// pair. Modern (post-2008) records have these populated; older records
// use the legacy `occurrences` table which we don't ingest.
type Finding struct {
	EvID        string
	AircraftKey int
	FindingNo   int

	FindingCode        *string
	FindingDescription *string
	CauseFactor        *string // C = cause, F = factor, etc.
}
