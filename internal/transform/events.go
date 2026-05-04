package transform

import (
	"encoding/json"
	"fmt"

	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// Event takes a single CSV row (already advanced via Reader.Read) and
// returns a typed *domain.Event. It accumulates normalization counts
// into s for reporting at end of run.
//
// Returns an error only for unrecoverable cases (missing primary key).
// Bad data in non-key fields is logged via stats and nullified.
func Event(r *acsv.Reader, s *stats.Counter) (*domain.Event, error) {
	evID := acsv.StringRequired(r.Get("ev_id"))
	if evID == "" {
		s.Inc("events.skipped.missing_ev_id")
		return nil, fmt.Errorf("row missing ev_id")
	}

	// Coordinates: only set if both present and in valid range.
	lat, lon, coordsOK := acsv.CoordPair(
		r.Get("dec_latitude"),
		r.Get("dec_longitude"),
	)
	if coordsOK {
		s.Inc("events.coords.populated")
	} else {
		s.Inc("events.coords.missing_or_invalid")
	}

	// Build the raw JSONB. We drop METAR (large, redundant with decoded
	// weather columns) but keep everything else for debugging / future use.
	raw := r.Row()
	delete(raw, "metar")
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}

	ev := &domain.Event{
		EvID:   evID,
		NtsbNo: acsv.String(r.Get("ntsb_no")),
		EvType: acsv.String(r.Get("ev_type")),
		EvDate: acsv.Date(r.Get("ev_date")),
		EvTime: acsv.Int(r.Get("ev_time")),
		EvTmzn: acsv.String(r.Get("ev_tmzn")),

		EvCity:        acsv.String(r.Get("ev_city")),
		EvState:       acsv.String(r.Get("ev_state")),
		EvCountry:     acsv.String(r.Get("ev_country")),
		EvSiteZipcode: acsv.String(r.Get("ev_site_zipcode")),

		Latitude:  lat,
		Longitude: lon,

		LightCond:   acsv.String(r.Get("light_cond")),
		WxCondBasic: acsv.String(r.Get("wx_cond_basic")),

		EvHighestInjury: acsv.String(r.Get("ev_highest_injury")),
		InjTotF:         acsv.Int(r.Get("inj_tot_f")),
		InjTotS:         acsv.Int(r.Get("inj_tot_s")),
		InjTotM:         acsv.Int(r.Get("inj_tot_m")),
		InjTotN:         acsv.Int(r.Get("inj_tot_n")),
		InjTotT:         acsv.Int(r.Get("inj_tot_t")),

		MidAir:            acsv.Bool(r.Get("mid_air")),
		OnGroundCollision: acsv.Bool(r.Get("on_ground_collision")),

		Raw:            rawJSON,
		SourceLchgDate: acsv.Date(r.Get("lchg_date")),
	}

	if ev.EvDate == nil {
		s.Inc("events.dates.missing")
	}
	if ev.SourceLchgDate == nil {
		s.Inc("events.lchg.missing")
	}

	return ev, nil
}
