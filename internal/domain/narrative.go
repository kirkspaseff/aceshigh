package domain

// Narrative is the text content associated with an (event, aircraft) pair.
// All four narrative fields are nullable; many foreign / unprosecuted
// events have placeholder content like "import" or are entirely empty.
type Narrative struct {
	EvID        string
	AircraftKey int

	NarrAccp  *string // accident analysis
	NarrAccf  *string // factual narrative
	NarrCause *string // probable cause
	NarrInc   *string // incident narrative
}
