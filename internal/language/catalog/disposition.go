package catalog

import "fmt"

// Disposition is the closed admission policy of a construct Kind. It is the
// single authoritative owner of whether an inventoried occurrence of the kind
// is admissible: consumers switch exhaustively on it and never hardcode a
// per-kind rejection. The zero value DispositionInvalid is never valid;
// numDispositions is the terminal sentinel.
type Disposition uint8

const (
	DispositionInvalid Disposition = iota
	// DispositionActive is a live construct the pipeline is expected to handle.
	DispositionActive
	// DispositionDeprecated is a construct retained only for total toolchain
	// reconciliation; encountering one in real input is a rejected form.
	DispositionDeprecated
	// DispositionRecovery is a parser error-recovery construct (Bad*). It
	// exists only in trees produced by failed parses, which the loader already
	// rejects; encountering one in an admitted tree is a rejected form.
	DispositionRecovery

	// numDispositions is the terminal sentinel. It must remain last.
	numDispositions
)

var dispositionNames = [numDispositions]string{
	DispositionActive:     "active",
	DispositionDeprecated: "deprecated",
	DispositionRecovery:   "recovery",
}

// Valid reports whether d names a disposition in the catalog.
func (d Disposition) Valid() bool { return d > DispositionInvalid && d < numDispositions }

// String renders d for diagnostics.
func (d Disposition) String() string {
	if d.Valid() {
		return dispositionNames[d]
	}
	return fmt.Sprintf("catalog.Disposition(%d)", uint8(d))
}
