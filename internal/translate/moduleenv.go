// Module-environment assembly: language-ABI specifiers, co-generated
// package specifiers, external stub specifiers, and the static extern
// method tables, all relative to the module's directory.
package translate

import (
	"fmt"
	"go/token"
	"go/types"
	"path"
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
		entries := obligation.Methods()
		methods := make([]emit.ExternMethod, 0, len(entries))
		for _, entry := range entries {
			methods = append(methods, emit.ExternMethod{Name: entry.Method.Name(), Key: entry.Key, Slot: entry.Slot})
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
		module.BoxedComposites = append(module.BoxedComposites, emit.BoxedComposite{
			Canon: composite.Canon, T: composite.T, Eq: composite.Eq})
	}
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
			kept := methods[:0]
			for i := range methods {
				// Resolve the signature and check for withheld references
				// BEFORE any symbol is marked used, so a dropped member
				// leaves no import behind.
				refs, err := externMethodRefs(unit, context, sourceDir, obligation, methods[i].Key)
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
				forms, _, err := externAdapter(module, unit, context, sourceDir, obligation, module.ExternMethods[typeID][i].Key)
				if err != nil {
					return nil, err
				}
				module.ExternMethods[typeID][i].Adapter = forms.Adapter
				module.ExternMethods[typeID][i].AdapterType = forms.AdapterType
				module.ExternMethods[typeID][i].AdapterPtr = forms.AdapterPtr
				module.ExternMethods[typeID][i].AdapterPtrType = forms.AdapterPtrType
			}
		}
	}
	return module, nil
}

// externBoxedValue resolves the boxed VALUE payload type of an external
// named type — the single receiver rule shared with the union member
// spelling: a basic-underlying external type boxes its exact value
// carrier; every other external type boxes its branded handle. This is
// the one place the decision lives, so the stub receiver, the vtable
// adapters, and the union payloads can never diverge.
func externBoxedValue(context *packages.Package, sourceDir string, unit ir.Scope, obligation *ir.ExternTypeObligation, method *types.Func) (ir.Type, bool, error) {
	handle := ir.Type{Kind: ir.KindExternal, Go: obligation.Pkg + "." + obligation.Name, Named: obligation.Name, Pkg: obligation.Pkg}
	recvType := method.Type().(*types.Signature).Recv().Type()
	if pointer, isPointer := recvType.(*types.Pointer); isPointer {
		recvType = pointer.Elem()
	}
	named, isNamed := types.Unalias(recvType).(*types.Named)
	if !isNamed {
		return handle, false, nil
	}
	basic, isBasic := named.Underlying().(*types.Basic)
	if !isBasic {
		return handle, false, nil
	}
	carrier, err := ir.ResolveType(context, sourceDir, unit, basic, token.NoPos)
	if err != nil {
		return ir.Type{}, false, err
	}
	carrier.Go = obligation.Pkg + "." + obligation.Name
	carrier.Named = obligation.Name
	carrier.Pkg = obligation.Pkg
	return carrier, true, nil
}

// externAdapter spells one external method's typed vtable arrows: both
// the value-member and pointer-member forms, from the one boxed-value
// rule.
func externAdapter(module *emit.Module, unit ir.Scope, context *packages.Package, sourceDir string, obligation *ir.ExternTypeObligation, key string) (emit.ExternAdapterForms, []string, error) {
	entry, ok := obligation.MethodByKey(key)
	if !ok {
		return emit.ExternAdapterForms{}, nil, fmt.Errorf("GOTOTS_EXTERNAL_UNSUPPORTED: external type %s.%s has no recorded method for key %s",
			obligation.Pkg, obligation.Name, key)
	}
	method := entry.Method
	name := method.Name()
	signature := method.Type().(*types.Signature)
	value, basicCarrier, err := externBoxedValue(context, sourceDir, unit, obligation, method)
	if err != nil {
		return emit.ExternAdapterForms{}, nil, err
	}
	var params []ir.Var
	for i := range signature.Params().Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, signature.Params().At(i).Type(), token.NoPos)
		if err != nil {
			return emit.ExternAdapterForms{}, nil, err
		}
		params = append(params, ir.Var{Name: fmt.Sprintf("p%d", i), Type: t})
	}
	var results []ir.Type
	for i := range signature.Results().Len() {
		t, err := ir.ResolveType(context, sourceDir, unit, signature.Results().At(i).Type(), token.NoPos)
		if err != nil {
			return emit.ExternAdapterForms{}, nil, err
		}
		results = append(results, t)
	}
	callee, err := module.Symbol(obligation.Pkg, emit.ExternMethodSymbol(obligation.Name, name))
	if err != nil {
		return emit.ExternAdapterForms{}, nil, err
	}
	forms, err := emit.ExternAdapters(module, value, basicCarrier, params, results, callee)
	if err != nil {
		return emit.ExternAdapterForms{}, nil, err
	}
	var refs []string
	seen := map[string]bool{}
	for _, p := range params {
		collectPkgRefs(p.Type, seen, &refs)
	}
	for _, r := range results {
		collectPkgRefs(r, seen, &refs)
	}
	return forms, refs, nil
}

// collectPkgRefs gathers every package path an ir.Type's SPELLING
// requires to exist. An interface position contributes nothing: it spells
// as a locally-declared closed union whose members are FILTERED by
// availability (a non-materialized implementer is excluded from the
// spelled union and never imported), so an interface type can never make
// a method unavailable.
func collectPkgRefs(t ir.Type, seen map[string]bool, out *[]string) {
	if t.Kind == ir.KindIface {
		return
	}
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
}

// externMethodRefs resolves one external method's signature and returns
// the package paths it references, without touching the module.
func externMethodRefs(unit ir.Scope, context *packages.Package, sourceDir string, obligation *ir.ExternTypeObligation, key string) ([]string, error) {
	entry, ok := obligation.MethodByKey(key)
	if !ok {
		return nil, fmt.Errorf("GOTOTS_EXTERNAL_UNSUPPORTED: external type %s.%s has no recorded method for key %s",
			obligation.Pkg, obligation.Name, key)
	}
	signature := entry.Method.Type().(*types.Signature)
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
