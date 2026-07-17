// Package translate orchestrates the translation proof chain for one
// unit of packages: census-aligned declaration identity -> typed body IR
// -> conservative representation plan -> lowering -> deterministic
// TypeScript with provenance -> per-declaration proof records. Named
// types, calls, and pointer-receiver methods resolve across the unit
// through generated module imports.
//
// Every construct outside the reviewed subset fails closed with a stable
// GOTOTS_UNSUPPORTED_* diagnostic; nothing is approximated.
package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/abi"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
)

// Packages translates a set of loaded production packages as one unit:
// named types, direct calls, and pointer-receiver method calls resolve
// across every package in the set, and each package emits one module at
// core/<pkgPath>/package.ts importing its co-generated dependencies.
func Packages(pkgs []*packages.Package, sourceDir string, options Options) (*Generated, error) {
	sorted := append([]*packages.Package{}, pkgs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PkgPath < sorted[j].PkgPath })
	seen := map[string]bool{}
	paths := make([]string, 0, len(sorted))
	for _, p := range sorted {
		if seen[p.PkgPath] {
			return nil, fmt.Errorf("package %s appears twice in the translation unit", p.PkgPath)
		}
		seen[p.PkgPath] = true
		paths = append(paths, p.PkgPath)
	}
	unit := ir.NewScope(paths...)
	collectGenericInstances(unit, sorted)

	out := &Generated{
		Files:     map[string]string{},
		Ownership: map[string]string{},
		Withheld:  map[string]string{},
	}
	var emitters []func() error
	for _, p := range sorted {
		if err := translatePackage(out, p, sourceDir, unit, options, &emitters); err != nil {
			return nil, err
		}
	}
	// The withholding closure is a fixed point over the REAL emitted
	// imports (value, init, AND type-only), which are only known after a
	// module renders. Each round: emit every retained package (filtering
	// union members and references to already-withheld packages), then
	// withhold, by ONE level, any package whose refreshed imports still
	// reach a withheld package. Re-emission between levels drops the
	// droppable (union-member) references before they can over-withhold,
	// so only UNAVOIDABLE dependencies on withheld packages cascade. The
	// loop converges because withholding grows monotonically and the
	// retained set is finite.
	for {
		for _, emitPackage := range emitters {
			if err := emitPackage(); err != nil {
				return nil, err
			}
		}
		if !growWithheldByImports(out, sorted) {
			break
		}
	}
	if err := emitExternalStubs(out, unit, sorted[0], sourceDir, options); err != nil {
		return nil, err
	}
	// Every retained module must import only retained modules: a dangling
	// import into a withheld package is a closure defect, not tolerable
	// output.
	if err := assertRetainedImportsResolve(out); err != nil {
		return nil, err
	}

	abiFiles := abi.Files()
	abiNames := make([]string, 0, len(abiFiles))
	for name := range abiFiles {
		abiNames = append(abiNames, name)
	}
	sort.Strings(abiNames)
	for _, name := range abiNames {
		abiPath := path.Join(abiDir, name)
		content, err := emit.FileWithProvenance(emit.Provenance{
			SchemaVersion:  1,
			SourceRevision: options.SourceRevision,
			ProfileHash:    options.ProfileHash,
			AbiVersion:     abi.Version,
			Path:           abiPath,
		}, abiFiles[name])
		if err != nil {
			return nil, err
		}
		out.Files[abiPath] = content
		out.Ownership[abiPath] = "generated-language-abi"
	}

	sort.Slice(out.Proofs, func(i, j int) bool { return out.Proofs[i].ID < out.Proofs[j].ID })
	sort.Slice(out.Support, func(i, j int) bool { return out.Support[i].ID < out.Support[j].ID })
	// The single final evidence pass: every file and proof now exists
	// (core, ABI, external stubs), so stages finalize exactly once and
	// invalid evidence is a build failure, not a normalization.
	if err := finalizeEvidenceStages(out); err != nil {
		return nil, err
	}
	return out, nil
}

// fileSource pairs one production file's AST with its exact source.
type fileSource struct {
	file     *ast.File
	relative string
	source   []byte
}

// translatePackage translates one unit package into its generated module.
// Declarations are collected in two passes — types first, then functions
// and methods — so a method may precede its receiver type in file order.
func translatePackage(out *Generated, p *packages.Package, sourceDir string, unit ir.Scope, options Options, emitters *[]func() error) error {
	corePath := path.Join("core", p.PkgPath, "package.ts")

	var files []fileSource
	for _, file := range p.Syntax {
		filename := p.Fset.Position(file.Pos()).Filename
		relative, err := filepath.Rel(sourceDir, filename)
		if err != nil || strings.HasPrefix(relative, "..") {
			return fmt.Errorf("file %s is outside the source checkout", filename)
		}
		source, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		files = append(files, fileSource{file: file, relative: filepath.ToSlash(relative), source: source})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relative < files[j].relative })

	// Pass 1: type and variable declarations, plus fail-closed screening
	// of every other package-level GenDecl. An unsupported declaration
	// records its exact site (the unit becomes unimplemented and the
	// package's runnable output is withheld) without stopping analysis.
	structs := map[string]*ir.Struct{}
	var structOrder []string
	var packageVars []emit.PackageVar
	var carrierTypes []emit.CarrierType
	var ledger []BodySupport
	unimplementedUnits := 0
	declSite := func(id string, err error) bool {
		unsupported, ok := ir.AsUnsupported(err)
		if !ok {
			return false
		}
		ledger = append(ledger, BodySupport{
			ID: id, Package: p.PkgPath, State: ir.SupportUnimplemented,
			Sites: []ir.UnsupportedSite{{
				Code:      unsupported.Code,
				Class:     ir.ClassOf(unsupported),
				Construct: unsupported.Construct,
				Span:      unsupported.Span,
			}},
		})
		unimplementedUnits++
		return true
	}
	// The type checker's initialization order: each initialized variable
	// maps to its position; multi-variable initializers stay out.
	initOrder := map[*types.Var]int{}
	multiInit := map[*types.Var]bool{}
	for index, initializer := range p.TypesInfo.InitOrder {
		for _, lhs := range initializer.Lhs {
			if len(initializer.Lhs) > 1 {
				multiInit[lhs] = true
				continue
			}
			initOrder[lhs] = index
		}
	}

	for _, f := range files {
		for _, decl := range f.file.Decls {
			d, isGen := decl.(*ast.GenDecl)
			if !isGen {
				continue
			}
			switch d.Tok {
			case token.IMPORT:
				// Imports carry no semantics of their own — every use is
				// identity-checked against the unit — except blank imports,
				// whose only meaning is the imported package's init side
				// effects.
				for _, spec := range d.Specs {
					importSpec := spec.(*ast.ImportSpec)
					if importSpec.Name != nil && importSpec.Name.Name == "_" {
						position := p.Fset.Position(spec.Pos())
						declSite(goid.Repeatable(p.PkgPath, "import", "_", f.relative, position.Line, position.Column),
							&ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
								Construct: "blank import (init side effects)", Span: spanOf(p, sourceDir, spec.Pos())})
					}
				}
			case token.TYPE:
				for _, spec := range d.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					id := goid.TypeName(p.PkgPath, "type", typeSpec.Name.Name)
					if object, isType := p.TypesInfo.Defs[typeSpec.Name].(*types.TypeName); isType && object.IsAlias() {
						id = goid.TypeName(p.PkgPath, "alias", typeSpec.Name.Name)
					}
					object, hasDef := p.TypesInfo.Defs[typeSpec.Name].(*types.TypeName)
					if !hasDef {
						declSite(id, &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
							Construct: "type without typed definition", Span: spanOf(p, sourceDir, spec.Pos())})
						continue
					}
					if _, isStruct := object.Type().Underlying().(*types.Struct); !isStruct || object.IsAlias() {
						// A non-struct named type (or alias) erases to its
						// underlying carrier: uses spell the carrier type, and
						// its methods generate as package-level functions. A
						// non-alias named type still owns an rtti object for
						// interface use.
						carrier, underlying, err := erasedCarrier(p, sourceDir, unit, typeSpec, object)
						if err != nil {
							if declSite(id, err) {
								continue
							}
							return err
						}
						if !object.IsAlias() {
							carrierTypes = append(carrierTypes, emit.CarrierType{
								Name: typeSpec.Name.Name, Exported: typeSpec.Name.IsExported(),
								Underlying: underlying,
							})
						}
						aliasProof := Proof{
							ID: id, SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV2,
							Representations: map[string]string{"decl:" + typeSpec.Name.Name: "erased-to-carrier(" + carrier + ")"},
						}
						if object.IsAlias() {
							// An alias erases to its target at every use and
							// emits no declaration of its own.
							aliasProof.NoOutput = true
						} else {
							aliasProof.GeneratedFile = corePath
							aliasProof.GeneratedSymbol = typeSpec.Name.Name
						}
						out.Proofs = append(out.Proofs, aliasProof)
						continue
					}
					structDecl, err := ir.BuildStruct(p, sourceDir, unit, typeSpec, id)
					if err != nil {
						if declSite(id, err) {
							continue
						}
						return err
					}
					structs[structDecl.Name] = structDecl
					structOrder = append(structOrder, structDecl.Name)
					out.Proofs = append(out.Proofs, Proof{
						ID: id, SourceRevision: options.SourceRevision,
						Package: p.PkgPath, File: f.relative,
						LoweringPlan:    LoweringPlanV2,
						Representations: map[string]string{"decl:" + typeSpec.Name.Name: "class-direct-identity"},
						GeneratedFile:   corePath, GeneratedSymbol: structDecl.Name,
					})
				}
			case token.VAR:
				for _, spec := range d.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					if len(valueSpec.Values) != 0 && len(valueSpec.Values) != len(valueSpec.Names) {
						position := p.Fset.Position(spec.Pos())
						declSite(goid.Repeatable(p.PkgPath, "var", "multi", f.relative, position.Line, position.Column),
							&ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
								Construct: "package-level multi-value var initializer", Span: spanOf(p, sourceDir, spec.Pos())})
						continue
					}
					for i, name := range valueSpec.Names {
						variableID := goid.Value(p.PkgPath, "var", name.Name)
						if name.Name == "_" {
							position := p.Fset.Position(name.Pos())
							variableID = goid.Repeatable(p.PkgPath, "var", "_", f.relative, position.Line, position.Column)
						}
						object, hasDef := p.TypesInfo.Defs[name].(*types.Var)
						if !hasDef {
							declSite(variableID, &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
								Construct: "var without typed definition", Span: spanOf(p, sourceDir, name.Pos())})
							continue
						}
						if multiInit[object] {
							declSite(variableID, &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
								Construct: "package-level multi-variable initializer", Span: spanOf(p, sourceDir, name.Pos())})
							continue
						}
						if name.Name == "_" {
							if i >= len(valueSpec.Values) {
								// A blank variable without an initializer
								// has no effect and no binding, but its
								// census identity still needs an explicit
								// disposition: a proof of the (empty)
								// lowering.
								out.Proofs = append(out.Proofs, Proof{
									ID: variableID, SourceRevision: options.SourceRevision,
									Package: p.PkgPath, File: f.relative,
									LoweringPlan:    LoweringPlanV2,
									Representations: map[string]string{"decl:_": "blank-var(no-initializer, no output)"},
									NoOutput:        true,
								})
								continue
							}
						}
						t, err := ir.ResolveType(p, sourceDir, unit, object.Type(), name.Pos())
						if err != nil {
							if declSite(variableID, err) {
								continue
							}
							return err
						}
						var initExpr ast.Expr
						if i < len(valueSpec.Values) {
							initExpr = valueSpec.Values[i]
						}
						init, err := ir.BuildPackageVarInit(p, sourceDir, unit, initExpr, t, name.Pos())
						if err != nil {
							if declSite(variableID, err) {
								continue
							}
							return err
						}
						order := -1
						if position, has := initOrder[object]; has {
							order = position
						}
						if name.Name == "_" {
							// A blank variable's initializer still runs in
							// order; there is no binding to declare.
							blankHash, err := blankInitHash(p, f.source, initExpr)
							if err != nil {
								return err
							}
							packageVars = append(packageVars, emit.PackageVar{
								Name: "_", Type: t, Init: init, Order: order, Blank: true,
							})
							out.Proofs = append(out.Proofs, Proof{
								ID: variableID, SourceRevision: options.SourceRevision,
								Package: p.PkgPath, File: f.relative,
								LoweringPlan:    LoweringPlanV2,
								Representations: map[string]string{"decl:_": "ordered-effect(no binding)"},
								GeneratedFile:   corePath, GeneratedSymbol: "",
								EffectOnly: true,
								InitHash:   blankHash,
							})
							continue
						}
						packageVars = append(packageVars, emit.PackageVar{
							Name: name.Name, Type: t, Init: init, Exported: name.IsExported(), Order: order,
						})
						varInitHash := ""
						if initExpr != nil {
							start := p.Fset.Position(initExpr.Pos()).Offset
							end := p.Fset.Position(initExpr.End()).Offset
							if start < 0 || end > len(f.source) || start >= end {
								// A present initializer always spans a valid,
								// non-empty range; an invalid span is a hard
								// error, not silently absent evidence.
								return fmt.Errorf("var %s.%s: invalid initializer span [%d,%d) over %d bytes", p.PkgPath, name.Name, start, end, len(f.source))
							}
							digest := sha256.Sum256(f.source[start:end])
							varInitHash = hex.EncodeToString(digest[:])
						}
						out.Proofs = append(out.Proofs, Proof{
							ID: goid.Value(p.PkgPath, "var", name.Name), SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV2,
							Representations: map[string]string{"decl:" + name.Name: "module-let(live-binding," + conservativeCarrier(t) + ")"},
							GeneratedFile:   corePath, GeneratedSymbol: name.Name,
							InitHash: varInitHash,
						})
					}
				}
			case token.CONST:
				// Package-level constants generate no code: go/types folds
				// every use — including cross-package uses — into an exact
				// Const during body building, so the declaration's complete
				// disposition is fold-at-use.
				for _, spec := range d.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					for _, name := range valueSpec.Names {
						if name.Name == "_" {
							continue
						}
						out.Proofs = append(out.Proofs, Proof{
							ID: goid.Value(p.PkgPath, "const", name.Name), SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV2,
							Representations: map[string]string{"decl:" + name.Name: "const-folded-at-use"},
							NoOutput:        true,
						})
					}
				}
			default:
				position := p.Fset.Position(d.Pos())
				declSite(goid.Repeatable(p.PkgPath, "decl", d.Tok.String(), f.relative, position.Line, position.Column),
					&ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
						Construct: "package-level " + d.Tok.String() + " declaration", Span: spanOf(p, sourceDir, d.Pos())})
			}
		}
	}

	// Pass 2: functions and methods — attached to their receiver classes,
	// or standalone for receivers erased to carriers. Package init
	// functions take synthesized names and run at module evaluation in
	// file order, after every variable initializer — exactly Go.
	var functions []*ir.Func
	var carrierMethods []emit.Method
	var initCalls []string
	for _, f := range files {
		for _, decl := range f.file.Decls {
			funcDecl, isFunc := decl.(*ast.FuncDecl)
			if !isFunc {
				continue
			}
			function, proof, err := translateFunc(p, sourceDir, unit, f.relative, f.source, funcDecl, options)
			if err != nil {
				return err
			}
			if funcDecl.Recv == nil && funcDecl.Name.Name == "init" {
				function.Name = fmt.Sprintf("init$%d", len(initCalls))
				// The proof records the exact emitted symbol.
				proof.GeneratedSymbol = function.Name
				if function.Support == ir.SupportGenerated {
					initCalls = append(initCalls, function.Name)
				}
			}
			ledger = append(ledger, BodySupport{
				ID: function.ID, Package: p.PkgPath, State: function.Support, Sites: function.Sites,
			})
			if function.Support == ir.SupportGenerated {
				registry, err := supportRegistry()
				if err != nil {
					return err
				}
				for _, operation := range function.Operations {
					if !registry.Generated(operation) {
						return fmt.Errorf("GOTOTS_CENSUS_UNREGISTERED_CLASS:\ngenerated body %s uses operation class %q with no reviewed support decision in contracts/support-classes.json",
							function.ID, operation)
					}
				}
			}
			if function.Support == ir.SupportUnimplemented {
				// Every unsupported site is on the record; no runnable body
				// exists and the package's module is withheld below.
				unimplementedUnits++
				continue
			}
			proof.GeneratedFile = corePath
			out.Proofs = append(out.Proofs, *proof)
			if function.Receiver == nil {
				functions = append(functions, function)
				continue
			}
			owner := receiverBase(funcDecl.Recv)
			if owner == "" {
				return fmt.Errorf("method %s has no named receiver type in package %s", function.Name, p.PkgPath)
			}
			if structDecl, ok := structs[owner]; ok {
				structDecl.Methods = append(structDecl.Methods, function)
				continue
			}
			carrierMethods = append(carrierMethods, emit.Method{TypeName: owner, Fn: function})
		}
	}
	out.Support = append(out.Support, ledger...)
	// The function-literal ledger covers EVERY package — withheld ones
	// included: analysis dispositions exist regardless of emission.
	lits, err := packageFuncLits(p, sourceDir, files, ledger)
	if err != nil {
		return err
	}
	out.FuncLits = append(out.FuncLits, lits...)
	if unimplementedUnits > 0 {
		out.Withheld[p.PkgPath] = fmt.Sprintf("%d unimplemented units", unimplementedUnits)
		return nil
	}
	if len(functions) == 0 && len(structs) == 0 && len(carrierMethods) == 0 && len(packageVars) == 0 {
		// A package whose declarations are all compile-time (constants
		// fold at use sites) emits an empty module: dependents reference
		// no runtime symbol from it.
		return nil
	}

	// Emission is deferred until the complete withholding closure is
	// known: union aliases must reference only retained classes, and a
	// package withheld by the dependency cascade emits nothing.
	*emitters = append(*emitters, func() error {
		if _, withheld := out.Withheld[p.PkgPath]; withheld {
			return nil
		}
		return emitCorePackage(out, p, sourceDir, unit, options, corePath, files,
			functions, structs, structOrder, carrierMethods, carrierTypes, packageVars, initCalls)
	})
	return nil
}

// collectGenericInstances records every generic-function instantiation
// across the unit: the closed-world evidence that admits generic
// declarations.
func collectGenericInstances(unit ir.Scope, pkgs []*packages.Package) {
	// The dynamic-type universe: every named type declared in an owned
	// package, the closed-world set interface dispatch resolves over.
	for _, p := range pkgs {
		scope := p.Types.Scope()
		for _, name := range scope.Names() {
			if typeName, ok := scope.Lookup(name).(*types.TypeName); ok && !typeName.IsAlias() {
				if _, isNamed := typeName.Type().(*types.Named); isNamed {
					unit.AddConcreteType(typeName)
				}
			}
		}
	}
	for _, p := range pkgs {
		for ident, instance := range p.TypesInfo.Instances {
			args := make([]types.Type, 0, instance.TypeArgs.Len())
			for i := range instance.TypeArgs.Len() {
				args = append(args, instance.TypeArgs.At(i))
			}
			switch object := p.TypesInfo.Uses[ident].(type) {
			case *types.Func:
				unit.AddGenericInstance(object, args)
			case *types.TypeName:
				unit.AddGenericTypeInstance(object, args)
			}
		}
	}
}

// collectGenericInstances records every generic-function instantiation
// across the unit: the closed-world evidence that admits generic
// declarations.
