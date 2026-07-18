// A materialized placeholder renders its EXACT typed signature and a
// fail-closed goBodyUnimplemented throw — nothing else. The signature
// typechecks against callers; calling it fails closed with the body's
// identity. This is analysis materialization, never a wrong result.
package emit

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/ir"
)

func TestPlaceholderFunctionRendersSignatureAndThrow(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	fn := &ir.Func{
		ID: "example.com/p::func::F", Name: "F", Exported: true, Placeholder: true,
		Params:  []ir.Var{{Name: "x", Type: ir.Type{Kind: ir.KindString}}},
		Results: []ir.Var{{Type: ir.Type{Kind: ir.KindString}}},
	}
	var out strings.Builder
	if err := printFunc(&out, m, fn); err != nil {
		t.Fatalf("printFunc placeholder: %v", err)
	}
	got := out.String()
	// Exact typed signature preserved.
	if !strings.Contains(got, "export function F(x: string): string {") {
		t.Fatalf("placeholder signature missing/wrong:\n%s", got)
	}
	// Fail-closed throw carrying the body identity; never returns, so it
	// satisfies the string return type.
	if !strings.Contains(got, `gort$.goBodyUnimplemented("example.com/p::func::F");`) {
		t.Fatalf("placeholder throw missing:\n%s", got)
	}
}

func TestPlaceholderMethodRendersSignatureAndThrow(t *testing.T) {
	m := NewModule("example.com/p", "p", ABIImports{}, map[string]string{})
	method := &ir.Func{
		ID: "example.com/p::method::T::M", Name: "M", Exported: true, Placeholder: true,
		Receiver: &ir.Var{Name: "t", Type: ir.Type{Kind: ir.KindPointer, Go: "*T",
			Elem: &ir.Type{Kind: ir.KindStruct, Named: "T", Pkg: "example.com/p", Go: "T"}}},
		Results: []ir.Var{{Type: ir.Type{Kind: ir.KindBool}}},
	}
	var out strings.Builder
	if err := printMethodFunction(&out, m, "T", method); err != nil {
		t.Fatalf("printMethodFunction placeholder: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "export function T$M(") || !strings.Contains(got, "): boolean {") {
		t.Fatalf("placeholder method signature missing/wrong:\n%s", got)
	}
	if !strings.Contains(got, `gort$.goBodyUnimplemented("example.com/p::method::T::M");`) {
		t.Fatalf("placeholder method throw missing:\n%s", got)
	}
	// No receiver-clone preamble leaked into a placeholder body.
	if strings.Contains(got, "goClone$()") || strings.Contains(got, "goArrayClone") {
		t.Fatalf("placeholder method leaked receiver preamble:\n%s", got)
	}
}
