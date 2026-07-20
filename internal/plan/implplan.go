// The implementation-plan layer of the governing pipeline
// (numbered-order step 18): beside the region/requirement engine in
// this package (the value/storage half), every IMPLEMENTATION receives
// one complete typed plan derived from SEALED facts, and the emitter
// consumes plans — it never rediscovers semantics. Plans are atomic:
// one record per implementation, no parallel maps, no capability
// vectors, no partially overlapping decisions.
package plan

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/implid"
)

// Form is the ADR-0007 preference-order outcome for one
// implementation. The planner selects the FIRST exact form; every form
// past Parametric carries machine evidence for why earlier forms were
// insufficient.
type Form string

const (
	// FormParametric: ordinary parametric TypeScript, no operations.
	FormParametric Form = "parametric"
	// FormRepresentationOwned: the selected data representation owns
	// the needed operation (a Map owns key encoding).
	FormRepresentationOwned Form = "representation-owned"
	// FormConcreteAtSite: the leaf call site derives a direct concrete
	// operation.
	FormConcreteAtSite Form = "concrete-at-site"
	// FormSpecialized: one static specialization per actual binding
	// family, each with its own ImplementationID.
	FormSpecialized Form = "specialized"
	// FormException: a typed recorded exceptional mechanism.
	FormException Form = "exception"
	// FormManualRequired: outside principled automatic translation; a
	// typed manual-required disposition, never a silent placeholder.
	FormManualRequired Form = "manual-required"
)

// MethodEmission is the ADR-0006 nil-receiver lowering decision.
type MethodEmission string

const (
	MethodEmissionNotAMethod    MethodEmission = ""
	MethodOrdinary              MethodEmission = "ordinary"
	MethodOrdinaryNilChecked    MethodEmission = "ordinary-nil-checked"
	MethodFreeFunctionException MethodEmission = "free-function-exception"
)

// ImplementationPlan is one implementation's complete decision.
type ImplementationPlan struct {
	ID   implid.ID
	Form Form
	// InsufficiencyEvidence records, for every form past Parametric,
	// why each earlier form was not exact (machine-checkable keys;
	// required exactly when Form demands it).
	InsufficiencyEvidence []string
	// MethodEmission is the ADR-0006 outcome for methods.
	MethodEmission MethodEmission
}

// ImplBuilder accumulates implementation plans during planning.
type ImplBuilder struct {
	plans map[string]ImplementationPlan
}

func NewImplBuilder() *ImplBuilder {
	return &ImplBuilder{plans: map[string]ImplementationPlan{}}
}

// Put records one implementation's plan. A second plan for the same
// implementation, or a non-parametric form without insufficiency
// evidence, is a planner defect and panics — plans are atomic and
// evidence-bearing by construction.
func (b *ImplBuilder) Put(p ImplementationPlan) {
	key := p.ID.String()
	if _, exists := b.plans[key]; exists {
		panic("plan: implementation " + key + " planned twice — plans are atomic")
	}
	switch p.Form {
	case FormParametric, FormManualRequired:
	case FormRepresentationOwned, FormConcreteAtSite, FormSpecialized, FormException:
		if len(p.InsufficiencyEvidence) == 0 {
			panic("plan: " + key + " selects " + string(p.Form) + " without insufficiency evidence for the earlier forms")
		}
	default:
		panic("plan: " + key + " has unknown form " + string(p.Form))
	}
	b.plans[key] = p
}

// Build freezes the plan set for lowering.
func (b *ImplBuilder) Build() *ImplStore {
	return &ImplStore{plans: b.plans}
}

// ImplStore is the frozen plan set: read-only for lowering.
type ImplStore struct {
	plans map[string]ImplementationPlan
}

// Get reads one implementation's plan; a missing plan at lowering time
// is a totality defect the caller must fail closed on.
func (s *ImplStore) Get(id implid.ID) (ImplementationPlan, bool) {
	p, ok := s.plans[id.String()]
	return p, ok
}

// IDs returns every planned implementation, sorted — the lowering
// totality join runs against this set.
func (s *ImplStore) IDs() []string {
	out := make([]string, 0, len(s.plans))
	for key := range s.plans {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// Len reports the plan count (a named denominator input).
func (s *ImplStore) Len() int { return len(s.plans) }
