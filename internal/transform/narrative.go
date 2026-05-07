package transform

import (
	"fmt"

	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// Narrative converts one CSV row into a *domain.Narrative.
func Narrative(r *acsv.Reader, s *stats.Counter) (*domain.Narrative, error) {
	evID := acsv.StringRequired(r.Get("ev_id"))
	if evID == "" {
		s.Inc("narratives.skipped.missing_ev_id")
		return nil, fmt.Errorf("row missing ev_id")
	}
	keyPtr := acsv.Int(r.Get("aircraft_key"))
	if keyPtr == nil {
		s.Inc("narratives.skipped.missing_aircraft_key")
		return nil, fmt.Errorf("row missing aircraft_key (ev_id=%s)", evID)
	}

	n := &domain.Narrative{
		EvID:        evID,
		AircraftKey: *keyPtr,
		NarrAccp:    acsv.String(r.Get("narr_accp")),
		NarrAccf:    acsv.String(r.Get("narr_accf")),
		NarrCause:   acsv.String(r.Get("narr_cause")),
		NarrInc:     acsv.String(r.Get("narr_inc")),
	}

	// Track substantive vs placeholder content. Anything under ~50 chars
	// is almost certainly a placeholder ("import", "foreign", etc.) or
	// missing entirely.
	if n.NarrCause != nil && len(*n.NarrCause) > 50 {
		s.Inc("narratives.cause.substantive")
	} else {
		s.Inc("narratives.cause.empty_or_placeholder")
	}

	return n, nil
}
