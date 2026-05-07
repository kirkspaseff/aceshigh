package transform

import (
	"encoding/json"
	"fmt"

	acsv "github.com/kirkspaseff/aceshigh/internal/csv"
	"github.com/kirkspaseff/aceshigh/internal/domain"
	"github.com/kirkspaseff/aceshigh/internal/stats"
)

// Aircraft converts one CSV row into a *domain.Aircraft. Returns an error
// for unrecoverable cases (missing PK components); bad data in non-key
// fields is logged via stats and nullified.
//
// Note: the column name in the source CSV is "Aircraft_Key" (capital A, K)
// not "aircraft_key". mdb-export preserves the case from Access. Same
// quirk applies to "Aircraft_Key" wherever it appears.
func Aircraft(r *acsv.Reader, s *stats.Counter) (*domain.Aircraft, error) {
	evID := acsv.StringRequired(r.Get("ev_id"))
	if evID == "" {
		s.Inc("aircraft.skipped.missing_ev_id")
		return nil, fmt.Errorf("row missing ev_id")
	}

	keyPtr := acsv.Int(r.Get("Aircraft_Key"))
	if keyPtr == nil {
		s.Inc("aircraft.skipped.missing_aircraft_key")
		return nil, fmt.Errorf("row missing Aircraft_Key (ev_id=%s)", evID)
	}

	// Unmanned is BOOLEAN NOT NULL in the source. Treat any parse failure
	// as false so we never write NULL into a NOT NULL column.
	unmanned := false
	if u := acsv.Bool(r.Get("unmanned")); u != nil {
		unmanned = *u
	}

	raw := r.Row()
	rawJSON, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw: %w", err)
	}

	a := &domain.Aircraft{
		EvID:        evID,
		AircraftKey: *keyPtr,

		RegisNo:    acsv.String(r.Get("regis_no")),
		NtsbNoAcft: acsv.String(r.Get("ntsb_no")),

		AcftMake:     acsv.String(r.Get("acft_make")),
		AcftModel:    acsv.String(r.Get("acft_model")),
		AcftSeries:   acsv.String(r.Get("acft_series")),
		AcftYear:     acsv.Int(r.Get("acft_year")),
		AcftCategory: acsv.String(r.Get("acft_category")),
		Homebuilt:    acsv.Bool(r.Get("homebuilt")),
		Unmanned:     unmanned,

		FarPart: acsv.String(r.Get("far_part")),
		TypeFly: acsv.String(r.Get("type_fly")),

		Damage:   acsv.String(r.Get("damage")),
		AcftFire: acsv.String(r.Get("acft_fire")),
		AcftExpl: acsv.String(r.Get("acft_expl")),

		TotalSeats: acsv.Int(r.Get("total_seats")),
		NumEng:     acsv.Int(r.Get("num_eng")),

		OperName:    acsv.String(r.Get("oper_name")),
		OperCountry: acsv.String(r.Get("oper_country")),

		DprtAptID:   acsv.String(r.Get("dprt_apt_id")),
		DprtCity:    acsv.String(r.Get("dprt_city")),
		DprtState:   acsv.String(r.Get("dprt_state")),
		DprtCountry: acsv.String(r.Get("dprt_country")),
		DprtTime:    acsv.Int(r.Get("dprt_time")),

		DestAptID:   acsv.String(r.Get("dest_apt_id")),
		DestCity:    acsv.String(r.Get("dest_city")),
		DestState:   acsv.String(r.Get("dest_state")),
		DestCountry: acsv.String(r.Get("dest_country")),

		PhaseFltSpec: acsv.Int(r.Get("phase_flt_spec")),

		Raw: rawJSON,
	}

	if a.AcftMake == nil {
		s.Inc("aircraft.make.missing")
	}
	if a.FarPart == nil {
		s.Inc("aircraft.far_part.missing")
	}

	return a, nil
}
