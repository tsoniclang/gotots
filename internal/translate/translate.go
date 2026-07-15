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

// Proof is the translation proof record of one declaration.
type Proof struct {
	ID              string            `json:"id"`
	SourceRevision  string            `json:"sourceRevision,omitempty"`
	Package         string            `json:"package"`
	File            string            `json:"file"`
	SignatureHash   string            `json:"signatureHash"`
	BodyHash        string            `json:"bodyHash"`
	Operations      []string          `json:"operations"`
	Representations map[string]string `json:"representations"`
	LoweringPlan    string            `json:"loweringPlan"`
	GeneratedFile   string            `json:"generatedFile"`
	GeneratedSymbol string            `json:"generatedSymbol"`
}

// Generated is one deterministic translation result.
type Generated struct {
	// Files maps bundle-relative paths to complete file contents.
	Files map[string]string
	// Ownership maps every generated path to its ownership root.
	Ownership map[string]string
	Proofs    []Proof
}

// Options carries provenance inputs for product runs.
type Options struct {
	SourceRevision string
	ProfileHash    string
}

// LoweringPlanV1 names the conservative vertical-slice plan: every value
// uses the exact conservative carrier of its Go type; no direct-form
// optimizations are selected.
const LoweringPlanV1 = "conservative-v1"

// abiDir is the bundle directory of the language-ABI modules.
const abiDir = "language-abi"

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
	}
	for _, p := range sorted {
		if err := translatePackage(out, p, sourceDir, unit, options); err != nil {
			return nil, err
		}
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
	return out, nil
}

// translatePackage translates one unit package into its generated module.
// Declarations are collected in two passes — types first, then functions
// and methods — so a method may precede its receiver type in file order.
func translatePackage(out *Generated, p *packages.Package, sourceDir string, unit ir.Scope, options Options) error {
	corePath := path.Join("core", p.PkgPath, "package.ts")
	module, err := newModule(corePath, p.PkgPath, p.Types.Name(), unit)
	if err != nil {
		return err
	}

	type fileSource struct {
		file     *ast.File
		relative string
		source   []byte
	}
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

	// Pass 1: type and variable declarations, plus fail-closed screening
	// of every other package-level GenDecl.
	structs := map[string]*ir.Struct{}
	var structOrder []string
	var packageVars []emit.PackageVar
	var carrierTypes []emit.CarrierType
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
						return &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
							Construct: "blank import (init side effects)", Span: spanOf(p, sourceDir, spec.Pos())}
					}
				}
			case token.TYPE:
				for _, spec := range d.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					id := goid.TypeName(p.PkgPath, "type", typeSpec.Name.Name)
					object, hasDef := p.TypesInfo.Defs[typeSpec.Name].(*types.TypeName)
					if !hasDef {
						return &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
							Construct: "type without typed definition", Span: spanOf(p, sourceDir, spec.Pos())}
					}
					if _, isStruct := object.Type().Underlying().(*types.Struct); !isStruct || object.IsAlias() {
						// A non-struct named type (or alias) erases to its
						// underlying carrier: uses spell the carrier type, and
						// its methods generate as package-level functions. A
						// non-alias named type still owns an rtti object for
						// interface use.
						carrier, err := erasedCarrier(p, sourceDir, unit, typeSpec, object)
						if err != nil {
							return err
						}
						if !object.IsAlias() {
							carrierTypes = append(carrierTypes, emit.CarrierType{
								Name: typeSpec.Name.Name, Exported: typeSpec.Name.IsExported(),
							})
						}
						out.Proofs = append(out.Proofs, Proof{
							ID: id, SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV1,
							Representations: map[string]string{typeSpec.Name.Name: "erased-to-carrier(" + carrier + ")"},
							GeneratedFile:   corePath, GeneratedSymbol: "",
						})
						continue
					}
					structDecl, err := ir.BuildStruct(p, sourceDir, unit, typeSpec, id)
					if err != nil {
						return err
					}
					structs[structDecl.Name] = structDecl
					structOrder = append(structOrder, structDecl.Name)
					out.Proofs = append(out.Proofs, Proof{
						ID: id, SourceRevision: options.SourceRevision,
						Package: p.PkgPath, File: f.relative,
						LoweringPlan:    LoweringPlanV1,
						Representations: map[string]string{typeSpec.Name.Name: "class-direct-identity"},
						GeneratedFile:   corePath, GeneratedSymbol: structDecl.Name,
					})
				}
			case token.VAR:
				for _, spec := range d.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					if len(valueSpec.Values) != 0 && len(valueSpec.Values) != len(valueSpec.Names) {
						return &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
							Construct: "package-level multi-value var initializer", Span: spanOf(p, sourceDir, spec.Pos())}
					}
					for i, name := range valueSpec.Names {
						if name.Name == "_" {
							// Order-independent initializers make a blank
							// package variable effect-free.
							continue
						}
						object, hasDef := p.TypesInfo.Defs[name].(*types.Var)
						if !hasDef {
							return &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
								Construct: "var without typed definition", Span: spanOf(p, sourceDir, name.Pos())}
						}
						t, err := ir.ResolveType(p, sourceDir, unit, object.Type(), name.Pos())
						if err != nil {
							return err
						}
						var initExpr ast.Expr
						if i < len(valueSpec.Values) {
							initExpr = valueSpec.Values[i]
						}
						init, err := ir.BuildPackageVarInit(p, sourceDir, unit, initExpr, t, name.Pos())
						if err != nil {
							return err
						}
						packageVars = append(packageVars, emit.PackageVar{
							Name: name.Name, Type: t, Init: init, Exported: name.IsExported(),
						})
						out.Proofs = append(out.Proofs, Proof{
							ID: goid.Value(p.PkgPath, "var", name.Name), SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: f.relative,
							LoweringPlan:    LoweringPlanV1,
							Representations: map[string]string{name.Name: "module-let(live-binding," + conservativeCarrier(t) + ")"},
							GeneratedFile:   corePath, GeneratedSymbol: name.Name,
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
							LoweringPlan:    LoweringPlanV1,
							Representations: map[string]string{name.Name: "const-folded-at-use"},
							GeneratedFile:   corePath, GeneratedSymbol: "",
						})
					}
				}
			default:
				return &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
					Construct: "package-level " + d.Tok.String() + " declaration", Span: spanOf(p, sourceDir, d.Pos())}
			}
		}
	}

	// Pass 2: functions and methods — attached to their receiver classes,
	// or standalone for receivers erased to carriers.
	var functions []*ir.Func
	var carrierMethods []emit.Method
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
	if len(functions) == 0 && len(structs) == 0 && len(carrierMethods) == 0 && len(packageVars) == 0 {
		return fmt.Errorf("package %s has no translatable declarations", p.PkgPath)
	}

	structList := make([]*ir.Struct, 0, len(structOrder))
	for _, name := range structOrder {
		structList = append(structList, structs[name])
	}
	body, err := emit.Package(module, emit.Decls{
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
	return nil
}

// newModule builds the emission context of one generated module: the
// language-ABI specifiers plus one specifier per co-generated package,
// all relative to the module's own directory.
func newModule(corePath, pkgPath, pkgName string, unit ir.Scope) (*emit.Module, error) {
	fromDir := path.Dir(corePath)
	intsImport, err := relativeImport(fromDir, path.Join(abiDir, "goints.js"))
	if err != nil {
		return nil, err
	}
	runtimeImport, err := relativeImport(fromDir, path.Join(abiDir, "goruntime.js"))
	if err != nil {
		return nil, err
	}
	sliceImport, err := relativeImport(fromDir, path.Join(abiDir, "goslice.js"))
	if err != nil {
		return nil, err
	}
	ifaceImport, err := relativeImport(fromDir, path.Join(abiDir, "goiface.js"))
	if err != nil {
		return nil, err
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
	return emit.NewModule(pkgPath, pkgName, emit.ABIImports{
		Ints:    intsImport,
		Runtime: runtimeImport,
		Slice:   sliceImport,
		Iface:   ifaceImport,
	}, specifiers), nil
}

func translateFunc(p *packages.Package, sourceDir string, unit ir.Scope, relativeFile string, source []byte, decl *ast.FuncDecl, options Options) (*ir.Func, *Proof, error) {
	name := decl.Name.Name
	id := goid.Func(p.PkgPath, name)
	if decl.Recv != nil {
		id = goid.Method(p.PkgPath, receiverBase(decl.Recv), name)
	}

	bodyHash := ""
	if decl.Body != nil {
		start := p.Fset.Position(decl.Body.Pos()).Offset
		end := p.Fset.Position(decl.Body.End()).Offset
		if start < 0 || end > len(source) || start >= end {
			return nil, nil, fmt.Errorf("declaration %s has an invalid body span", id)
		}
		digest := sha256.Sum256(source[start:end])
		bodyHash = hex.EncodeToString(digest[:])
	}

	function, err := ir.BuildFunc(p, sourceDir, unit, decl, id, bodyHash)
	if err != nil {
		return nil, nil, err
	}

	object := p.TypesInfo.Defs[decl.Name].(*types.Func)
	signatureText := types.TypeString(object.Type(), func(pkg *types.Package) string { return pkg.Path() })
	signatureDigest := sha256.Sum256([]byte(signatureText))

	representations := map[string]string{}
	recordRepresentation := func(t ir.Type) {
		representations[t.Go] = conservativeCarrier(t)
	}
	if function.Receiver != nil {
		recordRepresentation(function.Receiver.Type)
	}
	for _, parameter := range function.Params {
		recordRepresentation(parameter.Type)
	}
	for _, result := range function.Results {
		recordRepresentation(result.Type)
	}

	return function, &Proof{
		ID:              id,
		SourceRevision:  options.SourceRevision,
		Package:         p.PkgPath,
		File:            relativeFile,
		SignatureHash:   hex.EncodeToString(signatureDigest[:]),
		BodyHash:        bodyHash,
		Operations:      function.Operations,
		Representations: representations,
		LoweringPlan:    LoweringPlanV1,
		GeneratedSymbol: name,
	}, nil
}

// collectGenericInstances records every generic-function instantiation
// across the unit: the closed-world evidence that admits generic
// declarations.
func collectGenericInstances(unit ir.Scope, pkgs []*packages.Package) {
	for _, p := range pkgs {
		for ident, instance := range p.TypesInfo.Instances {
			fn, isFunc := p.TypesInfo.Uses[ident].(*types.Func)
			if !isFunc {
				continue
			}
			args := make([]types.Type, 0, instance.TypeArgs.Len())
			for i := range instance.TypeArgs.Len() {
				args = append(args, instance.TypeArgs.At(i))
			}
			unit.AddGenericInstance(fn, args)
		}
	}
}

// erasedCarrier resolves the reviewed carrier of a non-struct named type
// or alias declaration; a type outside the subset fails closed.
func erasedCarrier(p *packages.Package, sourceDir string, unit ir.Scope, spec *ast.TypeSpec, object *types.TypeName) (string, error) {
	t, err := ir.ResolveType(p, sourceDir, unit, object.Type(), spec.Pos())
	if err != nil {
		return "", err
	}
	return conservativeCarrier(t), nil
}

// receiverBase names the receiver's base type from its AST.
func receiverBase(recv *ast.FieldList) string {
	if len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	for {
		switch base := t.(type) {
		case *ast.StarExpr:
			t = base.X
		case *ast.ParenExpr:
			t = base.X
		case *ast.Ident:
			return base.Name
		default:
			return ""
		}
	}
}

// conservativeCarrier names the exact conservative representation of one
// reviewed type under plan v1.
func conservativeCarrier(t ir.Type) string {
	switch {
	case t.Kind == ir.KindBool:
		return "boolean"
	case t.Kind == ir.KindString:
		return "js-string(equality-only-ordering-unsupported)"
	case t.Kind == ir.KindPointer:
		return "object-identity-nilable(undefined)"
	case t.Kind == ir.KindStruct:
		return "class-instance-value(copy-on-bind,in-place-store)"
	case t.Kind == ir.KindFunc:
		return "js-closure-nilable(undefined)-capture-by-reference"
	case t.Kind == ir.KindIface:
		return "iface-box-nilable(undefined)-rtti-identity"
	case t.Kind == ir.KindMap:
		return "js-map-nilable(undefined)-has-based-lookup"
	case t.Kind == ir.KindSlice:
		return "goslice-view-nilable(undefined)-conservative"
	case t.Kind.Float():
		return "number"
	case t.Kind.Wide64():
		return "bigint-exact-64"
	case t.Kind.Integer():
		return fmt.Sprintf("number-wrapped-%d", t.Kind.Bits())
	}
	return "unsupported"
}

func spanOf(p *packages.Package, sourceDir string, pos token.Pos) ir.Span {
	position := p.Fset.Position(pos)
	file := position.Filename
	if relative, err := filepath.Rel(sourceDir, file); err == nil && !strings.HasPrefix(relative, "..") {
		file = filepath.ToSlash(relative)
	}
	return ir.Span{File: file, Line: position.Line, Col: position.Column}
}

// relativeImport computes the ESM specifier from one bundle directory to a
// bundle path, always with an explicit ./ or ../ prefix.
func relativeImport(fromDir, to string) (string, error) {
	relative, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(to))
	if err != nil {
		return "", err
	}
	specifier := filepath.ToSlash(relative)
	if !strings.HasPrefix(specifier, ".") {
		specifier = "./" + specifier
	}
	return specifier, nil
}
