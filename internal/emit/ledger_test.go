// The emission ledger records at the REAL print sites: these tests run
// the actual emitters (not synthetic records) and assert the module
// ledger content, so reconcile's typed inputs are proven producer-real.
package emit

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestPackageEmissionRecordsBodyAndPlaceholderEvents(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	ordinary := &ir.Func{
		ID: "example.com/p::func::G", Name: "G", Exported: true,
		Body: &ir.Block{},
	}
	placeholder := &ir.Func{
		ID: "example.com/p::func::F", Name: "F", Exported: true, Placeholder: true,
		Results: []ir.Var{{Type: ir.Type{Kind: ir.KindString}}},
	}
	if _, _, _, err := Package(m, Decls{Functions: []*ir.Func{ordinary, placeholder}}); err != nil {
		t.Fatalf("Package: %v", err)
	}
	events := m.Emissions()
	counts := map[string]int{}
	for _, event := range events {
		counts[event.ID+"/"+string(event.Kind)]++
	}
	if counts["example.com/p::func::G/body"] != 1 {
		t.Fatalf("ordinary body event missing or duplicated: %+v", events)
	}
	if counts["example.com/p::func::F/body-placeholder"] != 1 {
		t.Fatalf("placeholder event missing or duplicated: %+v", events)
	}
	// A placeholder body must NOT also record an ordinary body event —
	// the double-count guard in emitTransactionalBody.
	if counts["example.com/p::func::F/body"] != 0 {
		t.Fatalf("placeholder double-recorded as body: %+v", events)
	}
	if len(events) != 2 {
		t.Fatalf("expected exactly 2 events, got %+v", events)
	}
}

func TestStubModuleRecordsExternSymbolsWithObligations(t *testing.T) {
	m := NewModule("example.com/x", "x", ABIImports{}, map[string]string{})
	if _, err := StubModule(m, []StubFunc{{ID: "example.com/x.Getenv", Name: "Getenv"}}, nil); err != nil {
		t.Fatalf("StubModule: %v", err)
	}
	records := m.ExternSymbols()
	if len(records) != 1 || records[0].Symbol != "Getenv" || records[0].Obligation != "example.com/x.Getenv" {
		t.Fatalf("extern symbol records = %+v", records)
	}
}

func TestOverlayAbandonDropsItsEvents(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	overlay := m.Overlay()
	overlay.recordEmission("example.com/p::func::Dead", EmissionBodyPlaceholder, "default")
	// Abandoned (never committed): the parent ledger must stay empty —
	// a failed print attempt never leaks emission evidence.
	if len(m.Emissions()) != 0 {
		t.Fatalf("abandoned overlay leaked events: %+v", m.Emissions())
	}
	overlay2 := m.Overlay()
	overlay2.recordEmission("example.com/p::func::Live", EmissionBody, "default")
	overlay2.Commit()
	if len(m.Emissions()) != 1 || m.Emissions()[0].ID != "example.com/p::func::Live" {
		t.Fatalf("committed overlay events lost: %+v", m.Emissions())
	}
}

// Family-variant struct clones share *ir.Func method pointers; their
// emissions must carry DISTINCT implementation keys derived from the
// printing context — the exact 71-artifact overwrite class of the
// baseline, now representable and therefore preventable.
func TestFamilyVariantEmissionsCarryDistinctImplementations(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	method := &ir.Func{
		ID: "example.com/p::method::T::M", Name: "M", Placeholder: true, MethodIdent: "M$0",
		Receiver: &ir.Var{Name: "t", Type: ir.Type{Kind: ir.KindPointer, Go: "*T",
			Elem: &ir.Type{Kind: ir.KindStruct, Named: "T", Pkg: "example.com/p", Go: "T"}}},
	}
	base := &ir.Struct{ID: "example.com/p::type::T", Name: "T", TypeParams: []string{"K"}, Methods: []*ir.Func{method}}
	clone := *base
	clone.FamilyEnc = true
	if _, _, artifacts, err := Package(m, Decls{Structs: []*ir.Struct{base, &clone}}); err != nil {
		t.Fatalf("Package: %v", err)
	} else {
		ids := map[string]bool{}
		for _, artifact := range artifacts {
			if ids[artifact.ImplementationID] {
				t.Fatalf("implementation collision in artifacts: %s", artifact.ImplementationID)
			}
			ids[artifact.ImplementationID] = true
		}
		if !ids["example.com/p::method::T::M/default"] || !ids["example.com/p::method::T::M/map-key-encoded"] {
			t.Fatalf("expected default + map-key-encoded implementations, got %v", ids)
		}
	}
	keys := map[string]bool{}
	for _, event := range m.Emissions() {
		keys[event.ID+"/"+event.Implementation] = true
	}
	if !keys["example.com/p::method::T::M/default"] || !keys["example.com/p::method::T::M/map-key-encoded"] {
		t.Fatalf("emission events lack distinct implementation keys: %v", keys)
	}
}
