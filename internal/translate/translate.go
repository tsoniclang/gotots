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
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/tsident"
	"github.com/tsoniclang/gotots/internal/typeid"
)

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
	var encVariants []string
	var ptrVariants []string
	var packageVars []emit.PackageVar
	var carrierTypes []emit.CarrierType
	var ledger []BodySupport
	unimplementedUnits := 0
	// declBlockers counts DECLARATION-level rejections (a type, variable, or
	// import whose shape could not be resolved). Unlike an unimplemented
	// BODY — whose exact signature is still known, so it materializes as a
	// typed throwing placeholder — a declaration blocker leaves a structural
	// hole (a missing class or binding) that dependents reference, so the
	// package cannot produce analyzable TypeScript at all. Materialization
	// eligibility is gated on declBlockers; publication is gated on any
	// unimplemented unit.
	declBlockers := 0
	declSite := func(id string, err error) bool {
		unsupported, ok := ir.AsUnsupported(err)
		if !ok {
			return false
		}
		ledger = append(ledger, BodySupport{
			ID: id, Package: p.PkgPath, Kind: "declaration", State: ir.SupportUnimplemented,
			Sites: []ir.UnsupportedSite{ir.SiteOf(unsupported)},
		})
		unimplementedUnits++
		declBlockers++
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
						importID := goid.Repeatable(p.PkgPath, "import", "_", f.relative, position.Line, position.Column)
						importPath, _ := strconv.Unquote(importSpec.Path.Value)
						if importPath == "embed" {
							// The Go spec defines "embed" as directive-only: it
							// declares no init effect at all (//go:embed
							// semantics live on the embedded VARIABLES, which
							// carry their own dispositions). The exact
							// translation of the blank import is nothing.
							out.Proofs = append(out.Proofs, Proof{
								ID: importID, SourceRevision: options.SourceRevision,
								Package: p.PkgPath, File: f.relative,
								LoweringPlan:    LoweringPlanV2,
								Representations: map[string]string{"decl:_": "blank-import(embed: directive-only, no output)"},
								NoOutput:        true,
							})
							continue
						}
						if unit.Owns(importPath) {
							// An owned target's init effects are carried by the
							// module's initialization edge (every owned source
							// import evaluates the imported module).
							out.Proofs = append(out.Proofs, Proof{
								ID: importID, SourceRevision: options.SourceRevision,
								Package: p.PkgPath, File: f.relative,
								LoweringPlan:    LoweringPlanV2,
								Representations: map[string]string{"decl:_": "blank-import(owned: module init edge, no output)"},
								NoOutput:        true,
							})
							continue
						}
						declSite(importID,
							&ir.Unsupported{Kind: ir.KindBlankImportInitSideEffects, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
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
						declSite(id, &ir.Unsupported{Kind: ir.KindTypeWithoutTypedDefinition, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
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
							carrier := emit.CarrierType{
								Name: typeSpec.Name.Name, Exported: typeSpec.Name.IsExported(),
								Underlying: underlying,
							}
							if namedType, isNamed := object.Type().(*types.Named); isNamed && namedType.TypeParams() != nil {
								for i := range namedType.TypeParams().Len() {
									carrier.TypeParams = append(carrier.TypeParams, namedType.TypeParams().At(i).Obj().Name())
								}
							}
							carrierTypes = append(carrierTypes, carrier)
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
							aliasShape, err := census.DeriveAliasShape(object, id)
							if err != nil {
								if declSite(id, err) {
									continue
								}
								return err
							}
							aliasProof.AliasShapeHash = shapeHash(census.AliasShapeSignature(aliasShape))
						} else {
							aliasProof.GeneratedFile = corePath
							aliasProof.GeneratedSymbol = tsident.EscapeDeclared(typeSpec.Name.Name)
							typeShape, err := census.DeriveTypeShape(object, id)
							if err != nil {
								if declSite(id, err) {
									continue
								}
								return err
							}
							aliasProof.TypeShapeHash = shapeHash(census.TypeShapeSignature(typeShape))
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
					if namedType, isNamed := object.Type().(*types.Named); isNamed && ir.HasEncFamilyInstances(unit, namedType) {
						// The "$ek" variant clones AFTER the walk (methods
						// attach to the base declaration later).
						encVariants = append(encVariants, structDecl.Name)
					}
					if namedType, isNamed := object.Type().(*types.Named); isNamed && ir.HasPtrCellInstances(unit, namedType) {
						ptrVariants = append(ptrVariants, structDecl.Name)
					}
					structShape, err := census.DeriveTypeShape(object, id)
					if err != nil {
						if declSite(id, err) {
							continue
						}
						return err
					}
					out.Proofs = append(out.Proofs, Proof{
						ID: id, SourceRevision: options.SourceRevision,
						Package: p.PkgPath, File: f.relative,
						LoweringPlan:    LoweringPlanV2,
						Representations: map[string]string{"decl:" + typeSpec.Name.Name: "class-direct-identity"},
						GeneratedFile:   corePath, GeneratedSymbol: tsident.EscapeDeclared(structDecl.Name),
						TypeShapeHash: shapeHash(census.TypeShapeSignature(structShape)),
					})
				}
			case token.VAR:
				for _, spec := range d.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					if len(valueSpec.Values) != 0 && len(valueSpec.Values) != len(valueSpec.Names) {
						position := p.Fset.Position(spec.Pos())
						declSite(goid.Repeatable(p.PkgPath, "var", "multi", f.relative, position.Line, position.Column),
							&ir.Unsupported{Kind: ir.KindPackageLevelMultiValueVarInitializer, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
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
							declSite(variableID, &ir.Unsupported{Kind: ir.KindVarWithoutTypedDefinition, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
								Construct: "var without typed definition", Span: spanOf(p, sourceDir, name.Pos())})
							continue
						}
						if multiInit[object] {
							declSite(variableID, &ir.Unsupported{Kind: ir.KindPackageLevelMultiVariableInitializer, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
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
						placeholderVar := false
						if err != nil {
							// The variable's TYPE resolved; only its INITIALIZER
							// is a recorded unsupported construct. The binding
							// declares with its exact type and zero value and the
							// initializer's order slot becomes a typed throwing
							// placeholder — the declaration surface stays whole,
							// so the package MATERIALIZES (publication-withheld)
							// instead of structurally blocking its dependents.
							unsupported, isTyped := ir.AsUnsupported(err)
							if !isTyped {
								return err
							}
							ledger = append(ledger, BodySupport{
								ID: variableID, Package: p.PkgPath, Kind: "initializer", State: ir.SupportUnimplemented,
								Sites: []ir.UnsupportedSite{ir.SiteOf(unsupported)},
							})
							unimplementedUnits++
							placeholderVar = true
							init = nil
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
								Placeholder: placeholderVar, ID: variableID,
							})
							if placeholderVar {
								// The unimplemented ledger entry IS the
								// disposition; a placeholder never also proves.
								continue
							}
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
							Placeholder: placeholderVar, ID: variableID,
						})
						if placeholderVar {
							// The unimplemented ledger entry IS the disposition;
							// a placeholder never also proves.
							continue
						}
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
						varTypeSpelling, err := census.DeriveVarType(object)
						if err != nil {
							return fmt.Errorf("var %s.%s: %w", p.PkgPath, name.Name, err)
						}
						out.Proofs = append(out.Proofs, Proof{
							ID: goid.Value(p.PkgPath, "var", name.Name), SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV2,
							Representations: map[string]string{"decl:" + name.Name: "module-let(live-binding," + conservativeCarrier(t) + ")"},
							GeneratedFile:   corePath, GeneratedSymbol: tsident.EscapeDeclared(name.Name),
							InitHash:    varInitHash,
							VarTypeHash: shapeHash(varTypeSpelling),
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
						// The constant's declaration-shape hash: the type the
						// translator resolved and the exact folded value, joined
						// against the census constant shape by stage 05. A
						// constant whose type has no canonical identity fails
						// closed rather than emitting hashless (unjoinable)
						// evidence.
						constObj, ok := p.TypesInfo.Defs[name].(*types.Const)
						if !ok {
							return fmt.Errorf("constant %s.%s has no typed definition", p.PkgPath, name.Name)
						}
						constType, err := typeid.Canonical(constObj.Type())
						if err != nil {
							return fmt.Errorf("constant %s.%s: %w", p.PkgPath, name.Name, err)
						}
						constSig := census.ConstShapeSignature(census.ConstShape{
							Type: constType, Value: constObj.Val().ExactString()})
						constDigest := sha256.Sum256([]byte(constSig))
						out.Proofs = append(out.Proofs, Proof{
							ID: goid.Value(p.PkgPath, "const", name.Name), SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV2,
							Representations: map[string]string{"decl:" + name.Name: "const-folded-at-use"},
							ConstHash:       hex.EncodeToString(constDigest[:]),
							NoOutput:        true,
						})
					}
				}
			default:
				position := p.Fset.Position(d.Pos())
				declSite(goid.Repeatable(p.PkgPath, "decl", d.Tok.String(), f.relative, position.Line, position.Column),
					&ir.Unsupported{Kind: ir.KindPackageLevel, Code: "GOTOTS_UNSUPPORTED_DECLARATION",
						Construct: "package-level " + d.Tok.String() + " declaration", Span: spanOf(p, sourceDir, d.Pos())})
			}
		}
	}

	functions, carrierMethods, initCalls, ledger, bodyUnimplemented, err := translateFunctionsPass(out, p, sourceDir, unit, options, files, structs, corePath, ledger)
	if err != nil {
		return err
	}
	unimplementedUnits += bodyUnimplemented
	out.Support = append(out.Support, ledger...)
	// The function-literal ledger covers EVERY package — withheld ones
	// included: analysis dispositions exist regardless of emission.
	lits, err := packageFuncLits(p, sourceDir, files, ledger)
	if err != nil {
		return err
	}
	out.FuncLits = append(out.FuncLits, lits...)
	if declBlockers > 0 {
		// A declaration-level blocker leaves a structural hole (a missing
		// class or binding) that dependents reference, so the package cannot
		// produce analyzable TypeScript. It is neither materialized nor
		// published; the transitive materialization cascade extends this to
		// its dependents.
		reason := fmt.Sprintf("%d declaration blockers", declBlockers)
		out.NotMaterialized[p.PkgPath] = reason
		out.Withheld[p.PkgPath] = reason
		return nil
	}
	if unimplementedUnits > 0 {
		// Every declaration resolved; the unimplemented BODIES materialize as
		// typed throwing placeholders. The package is analyzable but withheld
		// from the runnable product.
		out.Withheld[p.PkgPath] = fmt.Sprintf("%d unimplemented bodies (materialized as placeholders)", unimplementedUnits)
	}
	if len(functions) == 0 && len(structs) == 0 && len(carrierMethods) == 0 && len(packageVars) == 0 &&
		len(carrierTypes) == 0 && len(initCalls) == 0 {
		// A package whose declarations are all compile-time (constants
		// fold at use sites) emits no module: dependents reference no
		// runtime symbol from it. A CARRIER TYPE (a named non-struct
		// type with its rtti) or an init function is a runtime symbol and
		// must emit — its proofs reference the module file. The typed
		// disposition is recorded here, at the decision site.
		out.ModuleDispositions = append(out.ModuleDispositions,
			ModuleDisposition{Package: p.PkgPath, State: "no-runtime-output",
				Reason: "all declarations are compile-time; constants fold at use sites"})
		return nil
	}

	// Materialize every package that is not structurally blocked. The
	// materialization cascade (over source imports) runs before the emitters,
	// so a package importing a NotMaterialized package emits nothing rather
	// than a dangling reference. Publication withholding is computed
	// separately, after emission, and does NOT remove the analyzable file.
	// Encoded-family variants: cloned now, with the fully attached method
	// set, appended after the base declarations.
	for _, name := range encVariants {
		if base, has := structs[name]; has {
			encClone := *base
			encClone.FamilyEnc = true
			structs[name+"$ek"] = &encClone
			structOrder = append(structOrder, name+"$ek")
		}
	}
	for _, name := range ptrVariants {
		if base, has := structs[name]; has {
			ptrClone := *base
			ptrClone.FamilyPtrCell = true
			structs[name+"$pc"] = &ptrClone
			structOrder = append(structOrder, name+"$pc")
		}
	}
	*emitters = append(*emitters, func() error {
		if _, blocked := out.NotMaterialized[p.PkgPath]; blocked {
			return nil
		}
		return emitCorePackage(out, p, sourceDir, unit, options, corePath, files,
			functions, structs, structOrder, carrierMethods, carrierTypes, packageVars, initCalls)
	})
	return nil
}
