package ir

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// checkSource type-checks a single-file Go source and returns its package,
// so tests can obtain REAL go/types objects (e.g. methods of generic types)
// rather than fragile hand-built signatures.
func checkSource(t *testing.T, src string) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check("example.com/x", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	return pkg
}

// A method whose signature mentions a free type parameter has no exact
// canonical identity: MethodKey (via typeid) fails closed. This is the
// canonicalization-failure the external-obligation contract must surface
// IMMEDIATELY — never as an in-band poison key deferred to downstream
// rejection.
// freeTypeParamMethod builds an external type Recv with a method M whose
// parameter is a FREE type parameter — so MethodKey (via typeid) has no
// binder and fails closed. It returns both the receiver named type and the
// method (AddExternalMethod needs the owner to compute the dispatch slot).
func freeTypeParamMethod(t *testing.T) (*types.Named, *types.Func) {
	t.Helper()
	pkg := types.NewPackage("example.com/x", "x")
	recvTN := types.NewTypeName(token.NoPos, pkg, "Recv", nil)
	recv := types.NewNamed(recvTN, types.NewStruct(nil, nil), nil)
	tpTN := types.NewTypeName(token.NoPos, pkg, "T", nil)
	tp := types.NewTypeParam(tpTN, types.NewInterfaceType(nil, nil))
	recvVar := types.NewVar(token.NoPos, pkg, "r", recv)
	param := types.NewVar(token.NoPos, pkg, "p", tp) // free: not in any binder scope
	sig := types.NewSignatureType(recvVar, nil, nil, types.NewTuple(param), nil, false)
	return recv, types.NewFunc(token.NoPos, pkg, "M", sig)
}

func plainMethod(t *testing.T, name string) (*types.Named, *types.Func) {
	t.Helper()
	pkg := types.NewPackage("example.com/x", "x")
	recvTN := types.NewTypeName(token.NoPos, pkg, "Recv", nil)
	recv := types.NewNamed(recvTN, types.NewStruct(nil, nil), nil)
	recvVar := types.NewVar(token.NoPos, pkg, "r", recv)
	sig := types.NewSignatureType(recvVar, nil, nil, types.NewTuple(), types.NewTuple(), false)
	return recv, types.NewFunc(token.NoPos, pkg, name, sig)
}

func TestAddExternalMethodFailsClosedOnUnresolvableIdentity(t *testing.T) {
	recv, bad := freeTypeParamMethod(t)
	// Sanity: the identity genuinely cannot be computed.
	if _, err := MethodKey(bad); err == nil {
		t.Fatal("MethodKey must fail for a method with a free type parameter")
	}

	s := NewScope("example.com/owned")
	err := s.AddExternalMethod(recv, bad)
	if err == nil {
		t.Fatal("AddExternalMethod must return the canonicalization error, not swallow it")
	}
	// Nothing recorded: no poison key, no partial obligation. The failure
	// propagates through the typed return, and no downstream stage must
	// remember to reject an in-band placeholder.
	if got := s.ExternalTypes(); len(got) != 0 {
		t.Fatalf("a failed AddExternalMethod recorded %d obligations; must record nothing", len(got))
	}
}

func TestAddExternalMethodRecordsByCanonicalKeyAndSlot(t *testing.T) {
	recv, good := plainMethod(t, "M")
	key, err := MethodKey(good)
	if err != nil {
		t.Fatalf("MethodKey for a plain method: %v", err)
	}
	wantSlot, err := MethodSlot(recv, good)
	if err != nil {
		t.Fatalf("MethodSlot for a plain method: %v", err)
	}

	s := NewScope("example.com/owned")
	if err := s.AddExternalMethod(recv, good); err != nil {
		t.Fatalf("AddExternalMethod for a resolvable method: %v", err)
	}
	obligations := s.ExternalTypes()
	if len(obligations) != 1 {
		t.Fatalf("expected one external type obligation, got %d", len(obligations))
	}
	entries := obligations[0].Methods()
	if len(entries) != 1 {
		t.Fatalf("expected one method entry, got %d", len(entries))
	}
	if entries[0].Key != key || entries[0].Method != good {
		t.Fatal("method not recorded under its canonical key")
	}
	// The dispatch slot is carried on the SAME atomic record, so emit keys
	// the box vtable by the SAME selector interface dispatch uses — never
	// re-derived from the bare name, and it cannot drift from key/method.
	if entries[0].Slot != wantSlot {
		t.Fatalf("recorded slot %q; want the canonical dispatch slot %q", entries[0].Slot, wantSlot)
	}
	// Looked up only by full canonical identity, never a bare-name/poison key.
	if _, ok := obligations[0].MethodByKey(key); !ok {
		t.Fatal("recorded method not retrievable by its canonical key")
	}
	if _, ok := obligations[0].MethodByKey("M"); ok {
		t.Fatal("method retrievable by bare name")
	}
	if _, ok := obligations[0].MethodByKey("\x00unresolved\x00M"); ok {
		t.Fatal("method retrievable by poison key")
	}
}

// A method of a generic type (its signature mentions the receiver's type
// parameter) is not representable as an external obligation — no single
// exact stub/adapter. The sole constructor rejects it up front via the
// representability check and records nothing, so no downstream consumer can
// silently skip it. MethodKey for such a method resolves (the receiver
// binds the parameter), proving representability — not an identity failure
// — is the rejector.
func TestAddExternalMethodRejectsMethodOfGenericType(t *testing.T) {
	pkg := checkSource(t, "package x\ntype Box[T any] struct{ v T }\nfunc (b Box[T]) Get() T { return b.v }\n")
	box := pkg.Scope().Lookup("Box").Type().(*types.Named)
	if box.NumMethods() != 1 {
		t.Fatalf("expected Box to have one method, got %d", box.NumMethods())
	}
	method := box.Method(0)

	// MethodKey resolves (the receiver binds T), so representability — not an
	// identity failure — is what rejects it.
	if _, err := MethodKey(method); err != nil {
		t.Fatalf("MethodKey should resolve for a method of a generic type: %v", err)
	}
	s := NewScope("example.com/owned")
	if err := s.AddExternalMethod(box, method); err == nil {
		t.Fatal("AddExternalMethod must reject a method of a generic type as unrepresentable")
	}
	if got := s.ExternalTypes(); len(got) != 0 {
		t.Fatalf("a rejected method recorded %d obligations; must record nothing", len(got))
	}
}
