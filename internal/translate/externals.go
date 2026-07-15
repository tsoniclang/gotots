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
	var order []string
	for _, fn := range unit.ExternalFuncs() {
		external := fn.Pkg().Path()
		if _, seen := byPackage[external]; !seen {
			order = append(order, external)
		}
		byPackage[external] = append(byPackage[external], fn)
	}
	sort.Strings(order)

	for _, external := range order {
		stubPath := path.Join("external-stubs", external, "package.ts")
		module, err := newModule(stubPath, external, byPackage[external][0].Pkg().Name(), ir.NewScope())
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
				Representations: map[string]string{fn.Name(): "external-stub(registry-fail-closed)"},
				GeneratedFile:   stubPath, GeneratedSymbol: fn.Name(),
			})
		}
		body, err := emit.StubModule(module, stubs)
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
