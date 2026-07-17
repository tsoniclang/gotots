// Function-declaration translation and module-environment helpers.
package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/goid"
	"github.com/tsoniclang/gotots/internal/ir"
	"github.com/tsoniclang/gotots/internal/typeid"
)

func translateFunc(p *packages.Package, sourceDir string, unit ir.Scope, relativeFile string, source []byte, decl *ast.FuncDecl, options Options) (*ir.Func, *Proof, error) {
	name := decl.Name.Name
	id := goid.Func(p.PkgPath, name)
	generatedSymbol := name
	if decl.Recv != nil {
		id = goid.Method(p.PkgPath, receiverBase(decl.Recv), name)
		// The emitted method function spells Type$Method.
		generatedSymbol = receiverBase(decl.Recv) + "$" + name
	} else if goid.IsRepeatable("func", name) {
		// init and blank functions repeat legally: their identities are
		// position-qualified, exactly like the census records them.
		position := p.Fset.Position(decl.Name.Pos())
		id = goid.Repeatable(p.PkgPath, "func", name, relativeFile, position.Line, position.Column)
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
	signatureText := typeid.Canonical(object.Type())
	signatureDigest := sha256.Sum256([]byte(signatureText))

	representations := map[string]string{}
	var representationErr error
	recordRepresentation := func(t ir.Type) {
		// The canonical identity is the ONLY carrier key: the Go spelling
		// is not a semantic identity (aliases and unexported members
		// collide), so an empty Canon fails closed rather than keying the
		// representation ledger by a spelling.
		if t.Canon == "" {
			if representationErr == nil {
				representationErr = fmt.Errorf("%s: carrier type %q has no canonical identity", id, t.Go)
			}
			return
		}
		representations["carrier:"+t.Canon] = conservativeCarrier(t)
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
	if representationErr != nil {
		return nil, nil, representationErr
	}
	// The planner's per-region slice selections are part of the proof:
	// the representation gate re-derives and compares them.
	for name, candidate := range function.SlicePlans {
		representations["slice-local:"+name] = candidate
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
		LoweringPlan:    LoweringPlanV2,
		GeneratedSymbol: generatedSymbol,
	}, nil
}

// erasedCarrier resolves the reviewed carrier of a non-struct named type
// or alias declaration; a type outside the subset fails closed.
func erasedCarrier(p *packages.Package, sourceDir string, unit ir.Scope, spec *ast.TypeSpec, object *types.TypeName) (string, ir.Type, error) {
	t, err := ir.ResolveType(p, sourceDir, unit, object.Type(), spec.Pos())
	if err != nil {
		return "", ir.Type{}, err
	}
	return conservativeCarrier(t), t, nil
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
		case *ast.IndexExpr:
			t = base.X
		case *ast.IndexListExpr:
			t = base.X
		case *ast.Ident:
			return base.Name
		default:
			return ""
		}
	}
}

// packageFuncLits derives the independent disposition of every
// production function literal in the package: canonical identity and
// body hash exactly as the census records them, state from the
// enclosing unit's outcome — an unsupported site within the literal's
// span, or a declaration-level rejection of the parent, is
// unimplemented; everything else is generated.
func packageFuncLits(p *packages.Package, sourceDir string, files []fileSource, ledger []BodySupport) ([]FuncLitSupport, error) {
	sitesByID := map[string][]ir.UnsupportedSite{}
	stateByID := map[string]ir.SupportState{}
	for _, support := range ledger {
		sitesByID[support.ID] = support.Sites
		stateByID[support.ID] = support.State
	}
	var out []FuncLitSupport
	for _, f := range files {
		for _, decl := range f.file.Decls {
			// Var groups attribute EACH VALUE expression to ITS OWN
			// variable's identity — never the group's first name.
			if gen, isGen := decl.(*ast.GenDecl); isGen && gen.Tok == token.VAR {
				for _, spec := range gen.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for i, value := range valueSpec.Values {
						if i >= len(valueSpec.Names) {
							break
						}
						name := valueSpec.Names[i]
						id := goid.Value(p.PkgPath, "var", name.Name)
						if name.Name == "_" {
							position := p.Fset.Position(name.Pos())
							id = goid.Repeatable(p.PkgPath, "var", "_", f.relative, position.Line, position.Column)
						}
						state, has := stateByID[id]
						if !has {
							state = ir.SupportUnimplemented
						}
						lits, err := collectLits(p, f, value, id, state, sitesByID[id])
						if err != nil {
							return nil, err
						}
						out = append(out, lits...)
					}
				}
				continue
			}
			parentID, parentState, parentSites := declOutcome(p, f.relative, decl, stateByID, sitesByID)
			if parentID == "" {
				continue
			}
			lits, err := collectLits(p, f, decl, parentID, parentState, parentSites)
			if err != nil {
				return nil, err
			}
			out = append(out, lits...)
		}
	}
	return out, nil
}

// collectLits walks one attributed region for function literals with
// exact source-span membership (offset containment, not line numbers).
func collectLits(p *packages.Package, f fileSource, node ast.Node, parentID string, parentState ir.SupportState, parentSites []ir.UnsupportedSite) ([]FuncLitSupport, error) {
	var out []FuncLitSupport
	var litErr error
	ast.Inspect(node, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		position := p.Fset.Position(lit.Pos())
		id := goid.Repeatable(p.PkgPath, "funclit", "", f.relative, position.Line, position.Column)
		// The span covers the WHOLE literal — the func keyword, parameter
		// and result lists, and the body — so a rejection anywhere in the
		// signature (an unsupported parameter or result type) is contained
		// by the literal and does not fail open to "generated".
		startPos := p.Fset.Position(lit.Pos())
		endPos := p.Fset.Position(lit.End())
		if startPos.Offset < 0 || endPos.Offset > len(f.source) || startPos.Offset >= endPos.Offset {
			// A function literal always spans a non-empty, in-bounds range;
			// an invalid span is a source/fileset mismatch and a hard error
			// at the producer — invalid evidence must never be constructed.
			if litErr == nil {
				litErr = fmt.Errorf("funclit %s: invalid source span [%d,%d) over %d bytes", id, startPos.Offset, endPos.Offset, len(f.source))
			}
			return false
		}
		digest := sha256.Sum256(f.source[startPos.Offset:endPos.Offset])
		bodyHash := hex.EncodeToString(digest[:])
		state := "generated"
		if parentState == ir.SupportUnimplemented {
			state = "unimplemented"
			// A parent with site records may still have literals outside
			// every unsupported span; only sites strictly inside the
			// literal's exact (line, column) range make IT unimplemented.
			// A parent with no sites (declaration-level rejection) blocks
			// everything inside it.
			if len(parentSites) > 0 {
				state = "generated"
				for _, site := range parentSites {
					if site.Span.File != f.relative {
						continue
					}
					afterStart := site.Span.Line > startPos.Line ||
						(site.Span.Line == startPos.Line && site.Span.Col >= startPos.Column)
					beforeEnd := site.Span.Line < endPos.Line ||
						(site.Span.Line == endPos.Line && site.Span.Col <= endPos.Column)
					if afterStart && beforeEnd {
						state = "unimplemented"
						break
					}
				}
			}
		}
		out = append(out, FuncLitSupport{ID: id, Parent: parentID, Package: p.PkgPath, BodyHash: bodyHash, State: state})
		return true
	})
	return out, litErr
}

// declOutcome names a declaration's canonical identity and support
// outcome for funclit-parent linkage. Declarations that carry no
// production funclits (types, consts, imports) return "".
func declOutcome(p *packages.Package, relativeFile string, decl ast.Decl, states map[string]ir.SupportState, sites map[string][]ir.UnsupportedSite) (string, ir.SupportState, []ir.UnsupportedSite) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Body == nil {
			return "", "", nil
		}
		id := goid.Func(p.PkgPath, d.Name.Name)
		if d.Recv != nil {
			id = goid.Method(p.PkgPath, receiverBase(d.Recv), d.Name.Name)
		} else if goid.IsRepeatable("func", d.Name.Name) {
			position := p.Fset.Position(d.Name.Pos())
			id = goid.Repeatable(p.PkgPath, "func", d.Name.Name, relativeFile, position.Line, position.Column)
		}
		state, has := states[id]
		if !has {
			// A body with no ledger record was rejected at declaration
			// level: everything inside is blocked.
			state = ir.SupportUnimplemented
		}
		return id, state, sites[id]
	}
	return "", "", nil
}
