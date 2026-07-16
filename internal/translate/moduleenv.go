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
func newModule(modulePath, pkgPath, pkgName string, unit ir.Scope, context *packages.Package, sourceDir string, withheld map[string]string, opaqueInterfaces bool) (*emit.Module, error) {
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
	for _, composite := range unit.BoxedComposites() {
		module.BoxedComposites = append(module.BoxedComposites, emit.BoxedComposite{Canon: composite.Canon, T: composite.T})
	}
	module.Withheld = func(pkg string) bool { _, is := withheld[pkg]; return is }
	module.ExternMethods = externMethods
	// The opacity choice must be settled BEFORE the vtable adapters below
	// spell any interface type: a stub module spells interfaces as the
	// opaque supertype and must register no implementer-union aliases.
	module.OpaqueInterfaces = opaqueInterfaces
	// Extern vtable adapters: exactly typed arrows over the stub
	// exports, spelled here where the obligations' declared signatures
	// are available. External types then participate in interface
	// dispatch with no registry and no erased types.
	if context != nil {
		for _, obligation := range unit.ExternalTypes() {
			typeID := obligation.Pkg + "." + obligation.Name
			methods := module.ExternMethods[typeID]
			kept := methods[:0]
			for i := range methods {
				// Resolve the signature and check for withheld references
				// BEFORE any symbol is marked used, so a dropped member
				// leaves no import behind.
				refs, err := externMethodRefs(unit, context, sourceDir, obligation, methods[i].Name)
				if err != nil {
					return nil, err
				}
				withheldRef := false
				for _, ref := range refs {
					if _, is := withheld[ref]; is {
						withheldRef = true
						break
					}
				}
				if withheldRef {
					continue
				}
				kept = append(kept, methods[i])
			}
			module.ExternMethods[typeID] = kept
			// Build adapters only for surviving members.
			for i := range module.ExternMethods[typeID] {
				adapter, adapterType, _, err := externAdapter(module, unit, context, sourceDir, obligation, module.ExternMethods[typeID][i].Name)
				if err != nil {
					return nil, err
				}
				module.ExternMethods[typeID][i].Adapter = adapter
				module.ExternMethods[typeID][i].AdapterType = adapterType
			}
		}
	}
	return module, nil
}

// externAdapter spells one external method's typed vtable arrow.
func externAdapter(module *emit.Module, unit ir.Scope, context *packages.Package, sourceDir string, obligation *ir.ExternTypeObligation, name string) (string, string, []string, error) {
	method := obligation.Methods[name]
	signature := method.Type().(*types.Signature)
	handle := ir.Type{Kind: ir.KindExternal, Go: obligation.Pkg + "." + obligation.Name, Named: obligation.Name, Pkg: obligation.Pkg}
	recv := ir.Type{Kind: ir.KindPointer, Go: "*" + handle.Go, Named: obligation.Name, Pkg: obligation.Pkg, Elem: &handle}
	params := []ir.Var{{Name: "recv", Type: recv}}
	for i := range signature.Params().Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, signature.Params().At(i).Type(), token.NoPos)
		if err != nil {
			return "", "", nil, err
		}
		params = append(params, ir.Var{Name: fmt.Sprintf("p%d", i), Type: t})
	}
	var results []ir.Type
	for i := range signature.Results().Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, signature.Results().At(i).Type(), token.NoPos)
		if err != nil {
			return "", "", nil, err
		}
		results = append(results, t)
	}
	callee, err := module.Symbol(obligation.Pkg, emit.ExternMethodSymbol(obligation.Name, name))
	if err != nil {
		return "", "", nil, err
	}
	adapter, err := emit.TypedAdapter(module, params, results, callee)
	if err != nil {
		return "", "", nil, err
	}
	adapterType, err := emit.TypedAdapterType(module, params, results)
	if err != nil {
		return "", "", nil, err
	}
	var refs []string
	seen := map[string]bool{}
	for _, p := range params {
		collectPkgRefs(p.Type, seen, &refs)
	}
	for _, r := range results {
		collectPkgRefs(r, seen, &refs)
	}
	return adapter, adapterType, refs, nil
}

// collectPkgRefs gathers every package path an ir.Type references.
func collectPkgRefs(t ir.Type, seen map[string]bool, out *[]string) {
	if t.Pkg != "" && !seen[t.Pkg] {
		seen[t.Pkg] = true
		*out = append(*out, t.Pkg)
	}
	if t.Elem != nil {
		collectPkgRefs(*t.Elem, seen, out)
	}
	if t.Key != nil {
		collectPkgRefs(*t.Key, seen, out)
	}
	for _, arg := range t.TypeArgs {
		collectPkgRefs(arg, seen, out)
	}
	for _, member := range t.IfaceMembers {
		if member.Pkg != "" && !seen[member.Pkg] {
			seen[member.Pkg] = true
			*out = append(*out, member.Pkg)
		}
	}
}

// externMethodRefs resolves one external method's signature and returns
// the package paths it references, without touching the module.
func externMethodRefs(unit ir.Scope, context *packages.Package, sourceDir string, obligation *ir.ExternTypeObligation, name string) ([]string, error) {
	signature := obligation.Methods[name].Type().(*types.Signature)
	var refs []string
	seen := map[string]bool{}
	add := func(goType types.Type) error {
		t, err := ir.ResolveType(context, sourceDir, unit, goType, token.NoPos)
		if err != nil {
			return err
		}
		collectPkgRefs(t, seen, &refs)
		return nil
	}
	for i := range signature.Params().Len() {
		if err := add(signature.Params().At(i).Type()); err != nil {
			return nil, err
		}
	}
	for i := range signature.Results().Len() {
		if err := add(signature.Results().At(i).Type()); err != nil {
			return nil, err
		}
	}
	return refs, nil
}
