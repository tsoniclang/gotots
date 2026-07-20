package plan

import (
	"github.com/tsoniclang/gotots/internal/facts"
	"github.com/tsoniclang/gotots/internal/implid"
)

// PlanMethodEmission is the ADR-0006 planning rule: sealed
// receiver-nilability facts become the method-emission decision the
// lowering consumes. The default is source shape; the free-function
// form is the recorded exception, never a family default.
//
//   - nil-tolerant methods keep the free-function exception (they are
//     semantically defined for nil receivers; an entry check would
//     change behavior);
//   - proven entry-equivalent methods emit as ordinary class methods
//     with a call-site nil check (elided later per-site by flow proof);
//   - unproven methods keep the free-function exception until the
//     analysis proves them — conservative-exact, and every analysis
//     refinement moves methods to source shape.
func PlanMethodEmission(fact facts.ReceiverNilability) MethodEmission {
	switch {
	case fact.ToleratesNil:
		return MethodFreeFunctionException
	case fact.EquivalentAtEntry:
		return MethodOrdinaryNilChecked
	default:
		return MethodFreeFunctionException
	}
}

// BuildMethodPlans derives one ImplementationPlan per method
// implementation from the sealed fact store. Methods missing a fact
// fail closed (the producer must be total over methods).
func BuildMethodPlans(builder *ImplBuilder, store *facts.Store, methods []implid.ID, delegateEligible map[string]bool) error {
	for _, id := range methods {
		fact, ok := store.ReceiverNilability(id.Source)
		if !ok {
			return &MissingFactError{Source: id.Source}
		}
		emission := PlanMethodEmission(fact)
		if emission == MethodFreeFunctionException && !delegateEligible[id.Source] {
			emission = MethodFreeFunctionNoDelegate
		}
		p := ImplementationPlan{ID: id, Form: FormParametric, MethodEmission: emission}
		if emission == MethodFreeFunctionException || emission == MethodFreeFunctionNoDelegate {
			p.Form = FormException
			reason := "adr-0006: receiver-nilability proof did not establish entry equivalence"
			if fact.ToleratesNil {
				reason = "adr-0006: method is semantically defined for nil receivers (B7 class)"
			}
			p.InsufficiencyEvidence = []string{reason}
		}
		builder.Put(p)
	}
	return nil
}

// MissingFactError reports a method without a nilability fact — a
// producer-totality defect.
type MissingFactError struct{ Source string }

func (e *MissingFactError) Error() string {
	return "plan: no receiver-nilability fact for " + e.Source + " (producer must be total over methods)"
}
