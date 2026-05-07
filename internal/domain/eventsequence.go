package domain

// EventSequence is one occurrence in the timeline of what happened during
// an event, for a given (event, aircraft). Modern records use this;
// older records use the legacy `occurrences` table which we don't ingest.
type EventSequence struct {
	EvID         string
	AircraftKey  int
	OccurrenceNo int

	OccurrenceCode        *string
	OccurrenceDescription *string
	PhaseNo               *string
	EventsoeNo            *string
	DefiningEv            bool // NOT NULL in source schema
}
