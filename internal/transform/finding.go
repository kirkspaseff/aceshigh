package transform

import (
	"fmt"

	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// Finding converts one CSV row into a *domain.Finding.
func Finding(r *acsv.Reader, s *stats.Counter) (*domain.Finding, error) {
	evID := acsv.StringRequired(r.Get("ev_id"))
	if evID == "" {
		s.Inc("findings.skipped.missing_ev_id")
		return nil, fmt.Errorf("row missing ev_id")
	}
	keyPtr := acsv.Int(r.Get("aircraft_key"))
	if keyPtr == nil {
		s.Inc("findings.skipped.missing_aircraft_key")
		return nil, fmt.Errorf("row missing aircraft_key (ev_id=%s)", evID)
	}
	noPtr := acsv.Int(r.Get("finding_no"))
	if noPtr == nil {
		s.Inc("findings.skipped.missing_finding_no")
		return nil, fmt.Errorf("row missing finding_no (ev_id=%s)", evID)
	}

	f := &domain.Finding{
		EvID:               evID,
		AircraftKey:        *keyPtr,
		FindingNo:          *noPtr,
		FindingCode:        acsv.String(r.Get("finding_code")),
		FindingDescription: acsv.String(r.Get("finding_description")),
		CauseFactor:        acsv.String(r.Get("cause_factor")),
	}

	if f.CauseFactor != nil {
		switch *f.CauseFactor {
		case "C":
			s.Inc("findings.cause_factor.cause")
		case "F":
			s.Inc("findings.cause_factor.factor")
		default:
			s.Inc("findings.cause_factor.other")
		}
	}

	return f, nil
}
