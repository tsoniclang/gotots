// Package translate orchestrates the translation proof chain for one
// package: census-aligned declaration identity -> typed body IR ->
// conservative representation plan -> lowering -> deterministic TypeScript
// with provenance -> per-declaration proof records.
//
// Version 1 covers the reviewed vertical-slice subset (exact integer,
// boolean, string-equality, float64, and control-flow semantics on plain
// functions). Every construct outside the subset fails closed with a
// stable GOTOTS_UNSUPPORTED_* diagnostic; nothing is approximated.
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

// Package translates one loaded production package.
func Package(p *packages.Package, sourceDir string, options Options) (*Generated, error) {
	out := &Generated{
		Files:     map[string]string{},
		Ownership: map[string]string{},
	}

	corePath := path.Join("core", p.PkgPath, "package.ts")
	abiDir := "language-abi"
	intsImport, err := relativeImport(path.Dir(corePath), path.Join(abiDir, "goints.js"))
	if err != nil {
		return nil, err
	}
	runtimeImport, err := relativeImport(path.Dir(corePath), path.Join(abiDir, "goruntime.js"))
	if err != nil {
		return nil, err
	}
	sliceImport, err := relativeImport(path.Dir(corePath), path.Join(abiDir, "goslice.js"))
	if err != nil {
		return nil, err
	}

	var functions []*ir.Func
	structs := map[string]*ir.Struct{}
	var structOrder []string
	for _, file := range p.Syntax {
		filename := p.Fset.Position(file.Pos()).Filename
		relative, err := filepath.Rel(sourceDir, filename)
		if err != nil || strings.HasPrefix(relative, "..") {
			return nil, fmt.Errorf("file %s is outside the source checkout", filename)
		}
		relative = filepath.ToSlash(relative)
		source, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				function, proof, err := translateFunc(p, sourceDir, relative, source, d, options)
				if err != nil {
					return nil, err
				}
				proof.GeneratedFile = corePath
				out.Proofs = append(out.Proofs, *proof)
				if function.Receiver == nil {
					functions = append(functions, function)
				} else {
					owner := function.Receiver.Type.Named
					structDecl, ok := structs[owner]
					if !ok {
						return nil, fmt.Errorf("method %s declared before its receiver type %s (multi-file ordering not resolved)", function.Name, owner)
					}
					structDecl.Methods = append(structDecl.Methods, function)
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.IMPORT:
					if len(d.Specs) > 0 {
						return nil, &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
							Construct: "package imports", Span: spanOf(p, sourceDir, d.Pos())}
					}
				case token.TYPE:
					for _, spec := range d.Specs {
						typeSpec := spec.(*ast.TypeSpec)
						id := goid.TypeName(p.PkgPath, "type", typeSpec.Name.Name)
						structDecl, err := ir.BuildStruct(p, sourceDir, typeSpec, id)
						if err != nil {
							return nil, err
						}
						structs[structDecl.Name] = structDecl
						structOrder = append(structOrder, structDecl.Name)
						out.Proofs = append(out.Proofs, Proof{
							ID: id, SourceRevision: options.SourceRevision,
							Package: p.PkgPath, File: relative,
							LoweringPlan:    LoweringPlanV1,
							Representations: map[string]string{typeSpec.Name.Name: "class-direct-identity"},
							GeneratedFile:   corePath, GeneratedSymbol: structDecl.Name,
						})
					}
				default:
					return nil, &ir.Unsupported{Code: "GOTOTS_UNSUPPORTED_DECLARATION",
						Construct: "package-level " + d.Tok.String() + " declaration", Span: spanOf(p, sourceDir, d.Pos())}
				}
			}
		}
	}
	if len(functions) == 0 && len(structs) == 0 {
		return nil, fmt.Errorf("package %s has no translatable declarations", p.PkgPath)
	}

	structList := make([]*ir.Struct, 0, len(structOrder))
	for _, name := range structOrder {
		structList = append(structList, structs[name])
	}
	body, err := emit.Package(structList, functions, intsImport, runtimeImport, sliceImport)
	if err != nil {
		return nil, err
	}
	coreContent, err := emit.FileWithProvenance(emit.Provenance{
		SchemaVersion:  1,
		SourceRevision: options.SourceRevision,
		ProfileHash:    options.ProfileHash,
		AbiVersion:     abi.Version,
		Path:           corePath,
	}, body)
	if err != nil {
		return nil, err
	}
	out.Files[corePath] = coreContent
	out.Ownership[corePath] = "generated-core"

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

func translateFunc(p *packages.Package, sourceDir, relativeFile string, source []byte, decl *ast.FuncDecl, options Options) (*ir.Func, *Proof, error) {
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

	function, err := ir.BuildFunc(p, sourceDir, decl, id, bodyHash)
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
