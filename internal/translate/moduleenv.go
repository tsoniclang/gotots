// Module-environment assembly: language-ABI specifiers, co-generated
// package specifiers, external stub specifiers, and the static extern
// method tables, all relative to the module's directory.
package translate

import (
	"path"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/ir"
)

// newModule builds the emission context of one generated module: the
// language-ABI specifiers plus one specifier per co-generated package,
// all relative to the module's own directory.
func newModule(modulePath, pkgPath, pkgName string, unit ir.Scope) (*emit.Module, error) {
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
	module.ExternMethods = externMethods
	return module, nil
}
