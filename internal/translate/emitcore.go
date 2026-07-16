// Core-module emission for one retained package: the module environment
// is built after the whole-unit withholding closure is known, so no
// retained output references a withheld class or module.
package translate

import (
	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/ir"
)

// emitCorePackage renders one retained package's module.
func emitCorePackage(out *Generated, p *packages.Package, sourceDir string, unit ir.Scope, options Options,
	corePath string, files []fileSource, functions []*ir.Func, structs map[string]*ir.Struct, structOrder []string,
	carrierMethods []emit.Method, carrierTypes []emit.CarrierType, packageVars []emit.PackageVar, initCalls []string) error {
	_ = files
	structList := make([]*ir.Struct, 0, len(structOrder))
	for _, name := range structOrder {
		structList = append(structList, structs[name])
	}
	// Synthesized anonymous-struct classes the package's bodies use.
	structList = append(structList, unit.AnonStructs(p.PkgPath)...)
	// The module's import environment is built after the declaration
	// passes so external obligations discovered while building bodies
	// resolve to their stub modules.
	module, err := newModule(corePath, p.PkgPath, p.Types.Name(), unit, p, sourceDir, out.Withheld)
	if err != nil {
		return err
	}
	// Every co-generated package this package imports is an
	// initialization edge: its module must evaluate even when constant
	// folding or type-only use erased every symbol reference.
	for importPath := range p.Imports {
		if unit.Owns(importPath) {
			module.RequireInitEdge(importPath)
		}
	}
	body, err := emit.Package(module, emit.Decls{
		InitCalls:    initCalls,
		Structs:      structList,
		Methods:      carrierMethods,
		CarrierTypes: carrierTypes,
		Vars:         packageVars,
		Functions:    functions,
	})
	if err != nil {
		return err
	}
	coreContent, err := emit.FileWithProvenance(emit.Provenance{
		SchemaVersion:  1,
		SourceRevision: options.SourceRevision,
		ProfileHash:    options.ProfileHash,
		AbiVersion:     abi.Version,
		Path:           corePath,
	}, body)
	if err != nil {
		return err
	}
	out.Files[corePath] = coreContent
	out.Ownership[corePath] = "generated-core"
	if out.ModuleImports == nil {
		out.ModuleImports = map[string][]string{}
	}
	out.ModuleImports[p.PkgPath] = module.CoGeneratedImports()
	return nil
}
