package plan

import (
	"testing"

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
