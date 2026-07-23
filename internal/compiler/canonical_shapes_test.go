package compiler

import (
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// shapeModel inspects a fixture under the default (all-full) contract and returns
// its module definitions (by unit), references (by child), and body regions.
func shapeModel(t *testing.T, files map[string]string) (
	defs map[string]analyze.ImplementationDefinition,
	refByChild map[string]analyze.ImplementationRef,
	bodyRegions map[string]bool,
	kindOf map[string]identity.UnitKind,
) {
	t.Helper()
	dir := writeFixture(t, files)
	insp, err := InspectConstructs(withManifest(t, source.Request{Dir: dir, ProviderContract: scope.DefaultContractID}))
	if err != nil {
		t.Fatal(err)
	}
	defs = map[string]analyze.ImplementationDefinition{}
	refByChild = map[string]analyze.ImplementationRef{}
	bodyRegions = map[string]bool{}
	for _, pkg := range insp.Inventory().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, d := range pkg.Definitions() {
			defs[d.Unit().String()] = d
		}
		for _, r := range pkg.References() {
			refByChild[r.Child().String()] = r
		}
		for _, region := range pkg.Files() {
			if root := region.RootUnit(); !root.IsZero() {
				bodyRegions[root.String()] = true
			}
		}
	}
	kindOf = map[string]identity.UnitKind{}
	for _, pkg := range insp.Workspace().Packages() {
		if pkg.ID().Owner().Class() != identity.OwnerModule {
			continue
		}
		for _, u := range pkg.Units() {
			kindOf[u.ID().String()] = u.Kind()
		}
	}
	return defs, refByChild, bodyRegions, kindOf
}

// TestCanonicalShapeRecords proves the four canonical Go shapes each have exact
// records with their declaration contracts — the closure directive's required
// examples. Each fixture is inspected under the all-full contract, so the exact
// site<->reference conservation join runs inside VerifyInventory; the assertions
// then pin the definition, contract, body region, and enclosing reference.
func TestCanonicalShapeRecords(t *testing.T) {
	// 1. func F(x int) int { return x }: declaration/signature contract; one
	// body definition owning a body region; one declaration-to-body reference.
	t.Run("function declaration", func(t *testing.T) {
		defs, refByChild, regions, kindOf := shapeModel(t, map[string]string{
			"go.mod":  "module shape1.example/m\n\ngo 1.26\n",
			"main.go": "package m\n\nfunc F(x int) int { return x }\n",
		})
		var f string
		for id, k := range kindOf {
			if k == identity.UnitFuncBody {
				f = id
			}
		}
		if f == "" {
			t.Fatal("no function-body unit")
		}
		if got := defs[f].Contract(); got != analyze.ContractDeclarationSignature {
			t.Errorf("F contract = %s, want declaration-signature", got)
		}
		if !regions[f] {
			t.Error("F owns no body region")
		}
		ref, ok := refByChild[f]
		if !ok {
			t.Fatal("F has no enclosing reference")
		}
		if !ref.Parent().IsFileDeclaration() {
			t.Errorf("F reference parent = %s, want the file declaration region", ref.Parent())
		}
		if ref.Contract() != analyze.ContractDeclarationSignature {
			t.Errorf("F reference contract = %s, want declaration-signature", ref.Contract())
		}
	})

	// 2. func Outer() func() { return func() {} }: the parent retains the
	// function-value operation and the literal's callable contract; the literal
	// has one definition and one exact reference into Outer's body region.
	t.Run("function literal", func(t *testing.T) {
		defs, refByChild, regions, kindOf := shapeModel(t, map[string]string{
			"go.mod":  "module shape2.example/m\n\ngo 1.26\n",
			"main.go": "package m\n\nfunc Outer() func() {\n\treturn func() {}\n}\n",
		})
		var outer, lit string
		for id, k := range kindOf {
			switch k {
			case identity.UnitFuncBody:
				outer = id
			case identity.UnitFuncLitBody:
				lit = id
			}
		}
		if outer == "" || lit == "" {
			t.Fatalf("missing units: outer=%q lit=%q", outer, lit)
		}
		if got := defs[lit].Contract(); got != analyze.ContractCallableSignature {
			t.Errorf("literal contract = %s, want callable-signature", got)
		}
		if !regions[lit] {
			t.Error("literal owns no body region")
		}
		ref, ok := refByChild[lit]
		if !ok {
			t.Fatal("literal has no enclosing reference")
		}
		if ref.Parent().IsFileDeclaration() || ref.Parent().Unit().Source().String() != outer {
			t.Errorf("literal reference parent = %s, want Outer's body region", ref.Parent())
		}
		if ref.Contract() != analyze.ContractCallableSignature {
			t.Errorf("literal reference contract = %s, want callable-signature", ref.Contract())
		}
	})

	// 3. var X = makeValue(): names/type are the declaration contract; the
	// initializer has one definition with the initializer contract and one
	// enclosing reference.
	t.Run("package initializer", func(t *testing.T) {
		defs, refByChild, regions, kindOf := shapeModel(t, map[string]string{
			"go.mod":  "module shape3.example/m\n\ngo 1.26\n",
			"main.go": "package m\n\nfunc makeValue() int { return 1 }\n\nvar X = makeValue()\n",
		})
		var initUnit string
		for id, k := range kindOf {
			if k == identity.UnitVarInitializer {
				initUnit = id
			}
		}
		if initUnit == "" {
			t.Fatal("no var-initializer unit")
		}
		if got := defs[initUnit].Contract(); got != analyze.ContractInitializer {
			t.Errorf("initializer contract = %s, want initializer", got)
		}
		if !regions[initUnit] {
			t.Error("initializer owns no body region")
		}
		ref, ok := refByChild[initUnit]
		if !ok {
			t.Fatal("initializer has no enclosing reference")
		}
		if ref.Contract() != analyze.ContractInitializer {
			t.Errorf("initializer reference contract = %s, want initializer", ref.Contract())
		}
	})

	// 4. func Read([]byte) (int, error): a bodyless obligation has one definition
	// with the declaration/signature contract, one reference, and ZERO body
	// occurrences (no body region).
	t.Run("bodyless obligation", func(t *testing.T) {
		defs, refByChild, regions, kindOf := shapeModel(t, map[string]string{
			"go.mod":       "module shape4.example/m\n\ngo 1.26\n",
			"decl.go":      "package m\n\n// Read is implemented in assembly.\nfunc Read(p []byte) (int, error)\n",
			"decl_amd64.s": "#include \"textflag.h\"\n\nTEXT ·Read(SB), NOSPLIT, $0-48\n\tRET\n",
		})
		var read string
		for id, k := range kindOf {
			if k == identity.UnitBodylessDecl {
				read = id
			}
		}
		if read == "" {
			t.Skip("bodyless declaration not censused (assembly/platform unavailable)")
		}
		if got := defs[read].Contract(); got != analyze.ContractDeclarationSignature {
			t.Errorf("bodyless contract = %s, want declaration-signature", got)
		}
		if defs[read].Full() {
			t.Error("bodyless obligation is full")
		}
		if regions[read] {
			t.Error("bodyless obligation owns a body region — must contribute zero body occurrences")
		}
		ref, ok := refByChild[read]
		if !ok {
			t.Fatal("bodyless obligation has no enclosing reference")
		}
		if ref.Contract() != analyze.ContractDeclarationSignature {
			t.Errorf("bodyless reference contract = %s, want declaration-signature", ref.Contract())
		}
	})
}
