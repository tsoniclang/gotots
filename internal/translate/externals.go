// External-contract stub generation: one typed stub module per external
// package with admitted obligations, delegating to the ABI registry by
// canonical identity and failing closed at runtime unless the emulation
// layer registered behavior.
package translate

import (
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
		module, err := newModule(stubPath, external, goName, unit)
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
				LoweringPlan:    LoweringPlanV1,
				Representations: map[string]string{fn.Name(): "external-stub(typed-static, fail-closed)"},
				GeneratedFile:   stubPath, GeneratedSymbol: fn.Name(),
			})
		}
		members := membersByPackage[external]
		sort.Slice(members, func(i, j int) bool { return members[i].Name < members[j].Name })
		for _, member := range members {
			out.Proofs = append(out.Proofs, Proof{
				ID: goid.Value(external, "extern-member", member.Name), SourceRevision: options.SourceRevision,
				Package:         external,
				LoweringPlan:    LoweringPlanV1,
				Representations: map[string]string{member.Name: "external-stub(typed-static, fail-closed)"},
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
	params := signature.Params()
	for i := range params.Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, params.At(i).Type(), token.NoPos)
		if err != nil {
			return emit.StubFunc{}, err
		}
		stub.Params = append(stub.Params, ir.Var{Name: params.At(i).Name(), Type: t})
	}
	results := signature.Results()
	for i := range results.Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, results.At(i).Type(), token.NoPos)
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
	names := make([]string, 0, len(obligation.Methods))
	for name := range obligation.Methods {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		method := obligation.Methods[name]
		signature := method.Type().(*types.Signature)
		// The receiver arrives as the handle a caller holds: through a
		// pointer it may be nil, and the implementation carries the
		// concrete method's exact nil semantics.
		recvType := ir.Type{Kind: ir.KindPointer, Go: "*" + typeID, Named: obligation.Name, Pkg: obligation.Pkg, Elem: &handle}
		member := emit.StubMember{
			ID:     typeID + "." + name,
			Name:   obligation.Name + "$" + name,
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
