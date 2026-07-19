// External-contract stub generation: one typed stub module per external
// package with admitted obligations, delegating to the ABI registry by
// canonical identity and failing closed at runtime unless the emulation
// layer registered behavior.
package translate

import (
	"fmt"
	"go/token"
	"go/types"
	"path"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/typeid"
)

// emitExternalStubs generates one typed stub module per external package
// with admitted contracts. Each stub delegates to the external-contract
// registry by canonical identity and fails closed at runtime unless the
// emulation layer registered behavior.
func emitExternalStubs(out *Generated, unit ir.Scope, context *packages.Package, sourceDir string, options Options) error {
	byPackage := map[string][]*types.Func{}
	packageNames := map[string]string{}
	var order []string
	record := func(external, goName string) {
		if _, seen := byPackage[external]; !seen {
			if _, named := packageNames[external]; !named {
				order = append(order, external)
			}
		}
		if goName != "" {
			packageNames[external] = goName
		}
	}
	for _, fn := range unit.ExternalFuncs() {
		record(fn.Pkg().Path(), fn.Pkg().Name())
		byPackage[fn.Pkg().Path()] = append(byPackage[fn.Pkg().Path()], fn)
	}
	// Type-member and variable obligations extend the per-package stub
	// surface (a package may contribute members without any function).
	membersByPackage := map[string][]emit.StubMember{}
	for _, obligation := range unit.ExternalTypes() {
		members, err := externTypeMembers(obligation, unit, context, sourceDir)
		if err != nil {
			return err
		}
		if _, seen := membersByPackage[obligation.Pkg]; !seen && len(byPackage[obligation.Pkg]) == 0 {
			order = append(order, obligation.Pkg)
		}
		membersByPackage[obligation.Pkg] = append(membersByPackage[obligation.Pkg], members...)
	}
	for _, external := range unit.ExternalVars() {
		dot := strings.LastIndex(external.ID, ".")
		pkg, name := external.ID[:dot], external.ID[dot+1:]
		t, err := ir.ResolveType(context, sourceDir, unit, external.Variable.Type(), token.NoPos)
		if err != nil {
			return err
		}
		if _, seen := membersByPackage[pkg]; !seen && len(byPackage[pkg]) == 0 {
			order = append(order, pkg)
		}
		membersByPackage[pkg] = append(membersByPackage[pkg], emit.StubMember{
			ID:         external.ID,
			Name:       name + "$get$",
			ResultType: &t,
		})
	}
	sort.Strings(order)

	for _, external := range order {
		stubPath := path.Join("external-stubs", external, "package.ts")
		goName := packageNames[external]
		if goName == "" {
			goName = path.Base(external)
		}
		module, err := newModule(stubPath, external, goName, unit, context, sourceDir, out.NotMaterialized)
		if err != nil {
			return err
		}
		var stubs []emit.StubFunc
		for _, fn := range byPackage[external] {
			stub, err := stubFor(fn, unit, context, sourceDir)
			if err != nil {
				return err
			}
			stubs = append(stubs, stub)
			out.Proofs = append(out.Proofs, Proof{
				ID: goid.Func(external, fn.Name()), SourceRevision: options.SourceRevision,
				Package:         external,
				LoweringPlan:    LoweringPlanV2,
				Representations: map[string]string{"decl:" + fn.Name(): "external-stub(typed-static, fail-closed)"},
				GeneratedFile:   stubPath, GeneratedSymbol: fn.Name(),
			})
		}
		members := membersByPackage[external]
		sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
		for _, member := range members {
			out.Proofs = append(out.Proofs, Proof{
				ID: goid.Value(external, "extern-member", member.Name), SourceRevision: options.SourceRevision,
				Package:         external,
				LoweringPlan:    LoweringPlanV2,
				Representations: map[string]string{"decl:" + member.Name: "external-stub(typed-static, fail-closed)"},
				GeneratedFile:   stubPath, GeneratedSymbol: member.Name,
			})
		}
		body, err := emit.StubModule(module, stubs, members)
		if err != nil {
			return err
		}
		content, err := emit.FileWithProvenance(emit.Provenance{
			SchemaVersion:  1,
			SourceRevision: options.SourceRevision,
			ProfileHash:    options.ProfileHash,
			AbiVersion:     abi.Version,
			Path:           stubPath,
		}, body)
		if err != nil {
			return err
		}
		out.Files[stubPath] = content
		out.Ownership[stubPath] = "generated-external-contracts"
	}
	return nil
}

// stubFor resolves one external function's declared signature into the
// stub's typed shape.
func stubFor(fn *types.Func, unit ir.Scope, context *packages.Package, sourceDir string) (emit.StubFunc, error) {
	signature := fn.Type().(*types.Signature)
	stub := emit.StubFunc{
		ID:   fn.Pkg().Path() + "." + fn.Name(),
		Name: fn.Name(),
	}
	typeParams := signature.TypeParams()
	for i := range typeParams.Len() {
		stub.TypeParams = append(stub.TypeParams, typeParams.At(i).Obj().Name())
	}
	binders := typeid.FuncBinders(signature)
	params := signature.Params()
	for i := range params.Len() {
		t, err := ir.ResolveTypeIn(context, sourceDir, unit, params.At(i).Type(), token.NoPos, binders)
		if err != nil {
			return emit.StubFunc{}, err
		}
		stub.Params = append(stub.Params, ir.Var{Name: params.At(i).Name(), Type: t})
	}
	results := signature.Results()
	for i := range results.Len() {
		t, err := ir.ResolveTypeIn(context, sourceDir, unit, results.At(i).Type(), token.NoPos, binders)
		if err != nil {
			return emit.StubFunc{}, err
		}
		stub.Results = append(stub.Results, ir.Var{Type: t})
	}
	return stub, nil
}

// externTypeMembers resolves one external type's stub surface: the
// value-semantics trio plus every unit-recorded method, all typed.
func externTypeMembers(obligation *ir.ExternTypeObligation, unit ir.Scope, context *packages.Package, sourceDir string) ([]emit.StubMember, error) {
	typeID := obligation.Pkg + "." + obligation.Name
	handle := ir.Type{Kind: ir.KindExternal, Go: typeID, Named: obligation.Name, Pkg: obligation.Pkg}
	out := []emit.StubMember{
		{ID: typeID + ".goZero$", Name: obligation.Name + "$goZero$", ResultType: &handle},
		{ID: typeID + ".goClone$", Name: obligation.Name + "$goClone$",
			Params: []ir.Var{{Name: "v", Type: handle}}, ResultType: &handle},
		{ID: typeID + ".goSet$", Name: obligation.Name + "$goSet$",
			Params: []ir.Var{{Name: "dst", Type: handle}, {Name: "src", Type: handle}}},
	}
	// Every recorded keyed-literal constructor obligation exports one
	// typed fail-closed stub: the emulation layer supplies the reviewed
	// construction.
	for _, shape := range obligation.LiteralShapes() {
		member := emit.StubMember{
			ID:         typeID + "$lit$" + strings.Join(shape.Fields, "$"),
			Name:       obligation.Name + "$lit$" + strings.Join(shape.Fields, "$") + "$",
			ResultType: &handle,
		}
		for i, field := range shape.Fields {
			member.Params = append(member.Params, ir.Var{Name: field, Type: shape.FieldTypes[i]})
		}
		out = append(out, member)
	}
	// Every referenced method (by canonical identity) contributes one stub
	// export. The emitted symbol derives from the display spelling; two
	// DISTINCT methods that would emit the same symbol (impossible for a
	// valid Go method set, which has unique names, but kept collision-safe
	// since the identities are canonical) fail closed rather than one
	// silently overwriting the other's export.
	emittedSymbols := map[string]string{}
	// Every recorded method is representable and carries its own canonical
	// key and slot (AddExternalMethod validated all three together), so this
	// loop emits ONE stub per record — never re-deriving the key and never
	// skipping a recorded method.
	for _, entry := range obligation.Methods() {
		method := entry.Method
		name := method.Name()
		signature := method.Type().(*types.Signature)
		symbol := obligation.Name + "$" + name
		if priorKey, seen := emittedSymbols[symbol]; seen && priorKey != entry.Key {
			return nil, fmt.Errorf("GOTOTS_EXTERNAL_UNSUPPORTED: external type %s.%s has two distinct methods emitting symbol %s",
				obligation.Pkg, obligation.Name, symbol)
		}
		emittedSymbols[symbol] = entry.Key
		// The receiver arrives as the boxed VALUE payload — the single
		// external-value rule: a basic-underlying external type's method
		// receives its exact value carrier (the adapters deref a pointer
		// member's cell with Go's nil panic BEFORE the call); every other
		// external type's method receives the handle, nilable through a
		// pointer, with the implementation carrying the concrete method's
		// exact nil semantics.
		value, basicCarrier, err := externBoxedValue(context, sourceDir, unit, obligation, method)
		if err != nil {
			return nil, err
		}
		recvType := value
		if !basicCarrier {
			recvType = ir.Type{Kind: ir.KindPointer, Go: "*" + typeID, Named: obligation.Name, Pkg: obligation.Pkg, Elem: &handle}
		}
		member := emit.StubMember{
			ID:     typeID + "." + name,
			Name:   symbol,
			Params: []ir.Var{{Name: "recv", Type: recvType}},
		}
		params := signature.Params()
		for i := range params.Len() {
			t, err := ir.ResolveType(context, sourceDir, unit, params.At(i).Type(), token.NoPos)
			if err != nil {
				return nil, err
			}
			member.Params = append(member.Params, ir.Var{Name: params.At(i).Name(), Type: t})
		}
		results := signature.Results()
		for i := range results.Len() {
			t, err := ir.ResolveType(context, sourceDir, unit, results.At(i).Type(), token.NoPos)
			if err != nil {
				return nil, err
			}
			member.Results = append(member.Results, t)
		}
		out = append(out, member)
	}
	return out, nil
}
