package plan

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/facts"
	"github.com/tsoniclang/gotots/internal/implid"
)

// Plans are atomic (no double planning) and evidence-bearing (a
// non-parametric form without insufficiency evidence is
// unrepresentable).
func TestImplPlanAtomicityAndEvidence(t *testing.T) {
	b := NewImplBuilder()
	id := implid.MustNew("p::func::F", "default")
	b.Put(ImplementationPlan{ID: id, Form: FormParametric})
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("double plan must panic")
			}
		}()
		b.Put(ImplementationPlan{ID: id, Form: FormParametric})
	}()
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("specialized without evidence must panic")
			}
		}()
		b.Put(ImplementationPlan{ID: implid.MustNew("p::func::G", "string-key"), Form: FormSpecialized})
	}()
	b.Put(ImplementationPlan{ID: implid.MustNew("p::func::G", "string-key"), Form: FormSpecialized,
		InsufficiencyEvidence: []string{"parametric: binding demands map-key encoding", "representation-owned: no map in scope", "concrete-at-site: multiple families"}})
	store := b.Build()
	if store.Len() != 2 || len(store.IDs()) != 2 {
		t.Fatalf("store = %d plans", store.Len())
	}
	if _, ok := store.Get(implid.MustNew("p::func::Missing", "default")); ok {
		t.Fatal("missing plan must report absent")
	}
}

// The ADR-0006 planning rule: tolerance and unproven methods take the
// recorded exception with evidence; proven methods emit ordinary with
// a check; a missing fact fails closed.
func TestMethodPlanningFromSealedFacts(t *testing.T) {
	store := facts.New()
	store.PutReceiverNilability("p::method::T::Proven", facts.ReceiverNilability{EquivalentAtEntry: true})
	store.PutReceiverNilability("p::method::T::Tolerant", facts.ReceiverNilability{ToleratesNil: true})
	store.PutReceiverNilability("p::method::T::Unproven", facts.ReceiverNilability{})
	store.Seal()
	builder := NewImplBuilder()
	methods := []implid.ID{
		implid.MustNew("p::method::T::Proven", "default"),
		implid.MustNew("p::method::T::Tolerant", "default"),
		implid.MustNew("p::method::T::Unproven", "default"),
	}
	if err := BuildMethodPlans(builder, store, methods, map[string]bool{"p::method::T::Proven": true, "p::method::T::Tolerant": true, "p::method::T::Unproven": true}); err != nil {
		t.Fatal(err)
	}
	plans := builder.Build()
	proven, _ := plans.Get(methods[0])
	if proven.MethodEmission != MethodOrdinaryNilChecked || proven.Form != FormParametric {
		t.Fatalf("proven = %+v", proven)
	}
	tolerant, _ := plans.Get(methods[1])
	if tolerant.MethodEmission != MethodFreeFunctionException || tolerant.Form != FormException || len(tolerant.InsufficiencyEvidence) == 0 {
		t.Fatalf("tolerant = %+v", tolerant)
	}
	unproven, _ := plans.Get(methods[2])
	if unproven.MethodEmission != MethodFreeFunctionException || len(unproven.InsufficiencyEvidence) == 0 {
		t.Fatalf("unproven = %+v", unproven)
	}
	// Missing fact fails closed.
	if err := BuildMethodPlans(NewImplBuilder(), store, []implid.ID{implid.MustNew("p::method::T::Ghost", "default")}, nil); err == nil {
		t.Fatal("missing fact must fail closed")
	}
}
