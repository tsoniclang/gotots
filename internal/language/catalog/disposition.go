package catalog

import "fmt"

// Disposition is the closed support disposition of a construct Kind: whether
// the catalog treats it as an active form to translate or a deprecated form a
// later phase must reject. The zero value DispositionInvalid is never valid;
// numDispositions is the terminal sentinel.
type Disposition uint8

const (
	DispositionInvalid Disposition = iota
	// DispositionActive is a live construct the pipeline is expected to handle.
	DispositionActive
	// DispositionDeprecated is a construct retained only for total toolchain
	// reconciliation; encountering one in real input is a rejected form.
	DispositionDeprecated

	// numDispositions is the terminal sentinel. It must remain last.
	numDispositions
)

var dispositionNames = [numDispositions]string{
	DispositionActive:     "active",
	DispositionDeprecated: "deprecated",
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
