package transform

import (
	"fmt"

	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// EventSequence converts one CSV row into a *domain.EventSequence.
func EventSequence(r *acsv.Reader, s *stats.Counter) (*domain.EventSequence, error) {
	evID := acsv.StringRequired(r.Get("ev_id"))
	if evID == "" {
		s.Inc("events_sequence.skipped.missing_ev_id")
		return nil, fmt.Errorf("row missing ev_id")
	}
	keyPtr := acsv.Int(r.Get("aircraft_key"))
	if keyPtr == nil {
		s.Inc("events_sequence.skipped.missing_aircraft_key")
		return nil, fmt.Errorf("row missing aircraft_key (ev_id=%s)", evID)
	}
	noPtr := acsv.Int(r.Get("occurrence_no"))
	if noPtr == nil {
		s.Inc("events_sequence.skipped.missing_occurrence_no")
		return nil, fmt.Errorf("row missing occurrence_no (ev_id=%s)", evID)
	}

	// defining_ev is BOOLEAN NOT NULL in source. Default to false on parse failure.
	defining := false
	if d := acsv.Bool(r.Get("defining_ev")); d != nil {
		defining = *d
	}

	e := &domain.EventSequence{
		EvID:                  evID,
		AircraftKey:           *keyPtr,
		OccurrenceNo:          *noPtr,
		OccurrenceCode:        acsv.String(r.Get("occurrence_code")),
		OccurrenceDescription: acsv.String(r.Get("occurrence_description")),
		PhaseNo:               acsv.String(r.Get("phase_no")),
		EventsoeNo:            acsv.String(r.Get("eventsoe_no")),
		DefiningEv:            defining,
	}

	if e.DefiningEv {
		s.Inc("events_sequence.defining")
	}

	return e, nil
}
