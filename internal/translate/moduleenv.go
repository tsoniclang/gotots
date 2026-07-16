// Module-environment assembly: language-ABI specifiers, co-generated
// package specifiers, external stub specifiers, and the static extern
// method tables, all relative to the module's directory.
package translate

import (
	"fmt"
	"go/token"
	"go/types"
	"path"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/ir"
)

// newModule builds the emission context of one generated module: the
// language-ABI specifiers plus one specifier per co-generated package,
// all relative to the module's own directory.
func newModule(modulePath, pkgPath, pkgName string, unit ir.Scope, context *packages.Package, sourceDir string, withheld map[string]string) (*emit.Module, error) {
	fromDir := path.Dir(modulePath)
	abiImports := emit.ABIImports{}
	for _, entry := range []struct {
		target *string
		file   string
	}{
		{&abiImports.Ints, "goints.js"},
		{&abiImports.Runtime, "goruntime.js"},
		{&abiImports.Slice, "goslice.js"},
		{&abiImports.Iface, "goiface.js"},
		{&abiImports.Extern, "goextern.js"},
	} {
		specifier, err := relativeImport(fromDir, path.Join(abiDir, entry.file))
		if err != nil {
			return nil, err
		}
		*entry.target = specifier
	}
	specifiers := map[string]string{}
	for _, other := range unit.Paths() {
		if other == pkgPath {
			continue
		}
		specifier, err := relativeImport(fromDir, path.Join("core", other, "package.js"))
		if err != nil {
			return nil, err
		}
		specifiers[other] = specifier
	}
	// Every admitted external contract resolves to its stub module.
	for _, fn := range unit.ExternalFuncs() {
		external := fn.Pkg().Path()
		if _, exists := specifiers[external]; exists {
			continue
		}
		specifier, err := relativeImport(fromDir, path.Join("external-stubs", external, "package.js"))
		if err != nil {
			return nil, err
		}
		specifiers[external] = specifier
	}
	// Every referenced external type resolves to its stub module, and its
	// recorded methods become the static dispatch tables of external rttis.
	externMethods := map[string][]emit.ExternMethod{}
	for _, obligation := range unit.ExternalTypes() {
		if _, exists := specifiers[obligation.Pkg]; !exists {
			specifier, err := relativeImport(fromDir, path.Join("external-stubs", obligation.Pkg, "package.js"))
			if err != nil {
				return nil, err
			}
			specifiers[obligation.Pkg] = specifier
		}
		names := make([]string, 0, len(obligation.Methods))
		for name := range obligation.Methods {
			names = append(names, name)
		}
		sort.Strings(names)
		methods := make([]emit.ExternMethod, 0, len(names))
		for _, name := range names {
			methods = append(methods, emit.ExternMethod{Name: name, Key: ir.MethodKey(obligation.Methods[name])})
		}
		externMethods[obligation.Pkg+"."+obligation.Name] = methods
	}
	for _, external := range unit.ExternalVars() {
		pkg := external.ID[:strings.LastIndex(external.ID, ".")]
		if _, exists := specifiers[pkg]; !exists {
			specifier, err := relativeImport(fromDir, path.Join("external-stubs", pkg, "package.js"))
			if err != nil {
				return nil, err
			}
			specifiers[pkg] = specifier
		}
	}
	module := emit.NewModule(pkgPath, pkgName, abiImports, specifiers)
	module.Withheld = func(pkg string) bool { _, is := withheld[pkg]; return is }
	module.ExternMethods = externMethods
	// Extern vtable adapters: exactly typed arrows over the stub
	// exports, spelled here where the obligations' declared signatures
	// are available. External types then participate in interface
	// dispatch with no registry and no erased types.
	if context != nil {
		for _, obligation := range unit.ExternalTypes() {
			typeID := obligation.Pkg + "." + obligation.Name
			methods := module.ExternMethods[typeID]
			for i := range methods {
				adapter, err := externAdapter(module, unit, context, sourceDir, obligation, methods[i].Name)
				if err != nil {
					return nil, err
				}
				methods[i].Adapter = adapter
			}
		}
	}
	return module, nil
}

// externAdapter spells one external method's typed vtable arrow.
func externAdapter(module *emit.Module, unit ir.Scope, context *packages.Package, sourceDir string, obligation *ir.ExternTypeObligation, name string) (string, error) {
	method := obligation.Methods[name]
	signature := method.Type().(*types.Signature)
	handle := ir.Type{Kind: ir.KindExternal, Go: obligation.Pkg + "." + obligation.Name, Named: obligation.Name, Pkg: obligation.Pkg}
	recv := ir.Type{Kind: ir.KindPointer, Go: "*" + handle.Go, Named: obligation.Name, Pkg: obligation.Pkg, Elem: &handle}
	params := []ir.Var{{Name: "recv", Type: recv}}
	for i := range signature.Params().Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, signature.Params().At(i).Type(), token.NoPos)
		if err != nil {
			return "", err
		}
		params = append(params, ir.Var{Name: fmt.Sprintf("p%d", i), Type: t})
	}
	var results []ir.Type
	for i := range signature.Results().Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, signature.Results().At(i).Type(), token.NoPos)
		if err != nil {
			return "", err
		}
		results = append(results, t)
	}
	callee, err := module.Symbol(obligation.Pkg, emit.ExternMethodSymbol(obligation.Name, name))
	if err != nil {
		return "", err
	}
	return emit.TypedAdapter(module, params, results, callee)
}
