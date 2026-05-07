package domain

import "encoding/json"

// Aircraft is one aircraft involved in an event, mapped to the aircraft table.
// Multi-aircraft events (e.g. midairs) have multiple Aircraft rows sharing
// the same EvID with distinct AircraftKey values.
type Aircraft struct {
	EvID        string // FK to events.ev_id
	AircraftKey int    // composite PK with EvID

	RegisNo    *string // tail number, e.g. "N530NA", "C-GBHZ"
	NtsbNoAcft *string // per-aircraft accident number suffix variant

	AcftMake     *string // raw, dirty: "PIPER" vs "Piper" etc.
	AcftModel    *string
	AcftSeries   *string
	AcftYear     *int
	AcftCategory *string // AIR | HELI | GLDR | ...
	Homebuilt    *bool
	Unmanned     bool // NOT NULL in source schema, defaults false

	FarPart *string // 091 | 121 | 135 | NUSC | NUSN
	TypeFly *string // PERS | INST | BUS | ...

	Damage   *string // DEST | SUBS | MINR | NONE
	AcftFire *string
	AcftExpl *string

	TotalSeats *int
	NumEng     *int

	OperName    *string
	OperCountry *string

	DprtAptID   *string
	DprtCity    *string
	DprtState   *string
	DprtCountry *string
	DprtTime    *int

	DestAptID   *string
	DestCity    *string
	DestState   *string
	DestCountry *string

	PhaseFltSpec *int // joins to code_lookups

	Raw json.RawMessage
}
