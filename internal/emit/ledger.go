// The emitter-owned emission ledger: every module records its own
// emission events AT the site that prints them, so no downstream
// consumer ever recovers emission facts from generated text. Records
// are appended per event — multiplicity is evidence (a duplicate
// identity is a visible defect, never collapsed by a set).
package emit

import (
	"github.com/tsoniclang/gotots/internal/implid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/plan"
)

// EmissionKind classifies one emission event in a generated module.
type EmissionKind string

const (
	// EmissionBody is a function or method body emitted in full.
	EmissionBody EmissionKind = "body"
	// EmissionBodyPlaceholder is a typed throwing body placeholder.
	EmissionBodyPlaceholder EmissionKind = "body-placeholder"
	// EmissionInitializer is a package-variable initializer slot emitted
	// in full inside the module initialization sequence.
	EmissionInitializer EmissionKind = "initializer"
	// EmissionInitializerPlaceholder is a typed throwing initializer slot.
	EmissionInitializerPlaceholder EmissionKind = "initializer-placeholder"
)

// EmissionEvent is one emission occurrence: the implementation identity
// and what was emitted for it. One identity emitting twice yields two
// events.
type EmissionEvent struct {
	ID   string
	Kind EmissionKind
	// Implementation is the specialization key of ADR-0010:
	// ImplementationID = ID + "/" + Implementation. Family-variant
	// emissions of one source declaration carry distinct keys, so
	// their multiplicity is explained, never a collision.
	Implementation string
}

// specializationKey derives the ADR-0010 key from the emission context.
func specializationKey(familyEnc, familyPtrCell bool) string {
	switch {
	case familyEnc:
		return "map-key-encoded"
	case familyPtrCell:
		return "pointer-cell"
	default:
		return "default"
	}
}

// ExternSymbolRecord is one exported symbol of an external-contract
// module, recorded where its export prints. A stub contract symbol
// carries the obligation identity its body throws; a support definition
// (union equality/key encoders, promoted delegates) carries none and is
// a direct implementation.
type ExternSymbolRecord struct {
	// Module is the owning module's package path: symbol identity is
	// Module + "::" + Symbol (Abs in math and Abs in path are distinct
	// symbols, never duplicates).
	Module string
	Symbol string
	// Obligation is the external obligation identity thrown by the stub
	// body, empty for directly implemented support definitions.
	Obligation string
}

func (m *Module) recordEmission(id string, kind EmissionKind, implementation string) {
	m.emissions = append(m.emissions, EmissionEvent{ID: id, Kind: kind, Implementation: implementation})
}

func (m *Module) recordExternSymbol(symbol, obligation string) {
	m.externSymbols = append(m.externSymbols, ExternSymbolRecord{Module: m.Pkg, Symbol: symbol, Obligation: obligation})
}

// Emissions exposes the module's emission events in emission order.
func (m *Module) Emissions() []EmissionEvent { return m.emissions }

// ExternSymbols exposes the module's exported-symbol records in
// emission order.
func (m *Module) ExternSymbols() []ExternSymbolRecord { return m.externSymbols }

// emissionsSince reports whether an event for id with the given kind was
// recorded at or after index from.
func (m *Module) emissionsSince(from int, id string, kind EmissionKind) bool {
	for _, event := range m.emissions[from:] {
		if event.ID == id && event.Kind == kind {
			return true
		}
	}
	return false
}

// MethodEmissionFor consults the frozen plan store for one method's
// ADR-0006 emission decision. A nil store (unit fixtures) is the
// zero-proof state: every method takes the conservative exception
// lowering, exactly as an all-unproven analysis would decide. A
// PRESENT store missing a method is a planner-totality defect.
func (m *Module) MethodEmissionFor(sourceID string) plan.MethodEmission {
	if m.MethodPlans == nil {
		return plan.MethodFreeFunctionException
	}
	p, ok := m.MethodPlans.Get(implid.MustNew(sourceID, "default"))
	if !ok {
		panic("emit: method " + sourceID + " has no implementation plan (planner must be total over methods)")
	}
	return p.MethodEmission
}

// methodPlanKey is the canonical plan-consult key of one method: the
// alias-resolved PlanKey when the builder recorded one, else the
// declaration ID (fixtures construct Funcs directly).
func methodPlanKey(method *ir.Func) string {
	if method.PlanKey != "" {
		return method.PlanKey
	}
	return method.ID
}
