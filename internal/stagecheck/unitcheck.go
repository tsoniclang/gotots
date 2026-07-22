package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/source"
)

// VerifyUnitCensus independently reconstructs the implementation-unit census
// and evidence-depth selection: for every finalized source file it re-parses
// the selected bytes (overlay-aware, streaming, its own file set, no
// typechecking), re-derives unit spans/kinds/hashes/C-dependence with its own
// walk, re-applies the shared provider contract to its own derived inputs,
// and exact-joins identity, hash, and depth against the producer — reporting
// both one-sided lists. No second type universe is created.
func VerifyUnitCensus(ws *source.Workspace, req source.Request, contract scope.ProviderContract) error {
	fail := func(reason string) error {
		return &VerificationError{Stage: "unit-census", Reason: reason}
	}
	var mismatches []string
	for _, pkg := range ws.Packages() {
		switch pkg.Disposition() {
		case source.DispositionBuiltinUniverse, source.DispositionUnsafeIntrinsic:
			continue // no censused units
		}
		producer := map[identity.SourceUnitID]source.SourceUnit{}
		for _, unit := range pkg.Units() {
			producer[unit.ID()] = unit
		}
		matched := map[identity.SourceUnitID]bool{}
		for _, file := range pkg.Files() {
			derived, err := deriveUnits(file, req.Overlay)
			if err != nil {
				return fail(err.Error())
			}
			for _, expected := range derived {
				got, exists := producer[expected.id]
				if !exists {
					mismatches = append(mismatches, "producer misses unit "+expected.id.String())
					continue
				}
				matched[expected.id] = true
				if got.Hash().String() != expected.hash {
					mismatches = append(mismatches, "unit "+expected.id.String()+" hash diverges")
				}
				expectedDepth, err := contract.UnitDepth(pkg.ID().Owner().Class(), pkg.Disposition(), expected.id.Kind(), expected.cDependent)
				if err != nil {
					return fail(err.Error())
				}
				if got.Depth() != expectedDepth {
					mismatches = append(mismatches, fmt.Sprintf("unit %s depth %s vs independently derived %s",
						expected.id, got.Depth(), expectedDepth))
				}
			}
		}
		for id := range producer {
			if !matched[id] {
				mismatches = append(mismatches, "producer holds unit "+id.String()+" the independent census does not derive")
			}
		}
	}
	if len(mismatches) > 0 {
		sort.Strings(mismatches)
		return fail(fmt.Sprintf("unit census join failed; %v", mismatches))
	}
	return nil
}

// derivedUnit is one independently derived census record.
type derivedUnit struct {
	id         identity.SourceUnitID
	hash       string
	cDependent bool
}

// deriveUnits re-parses one file's selected bytes and derives its top-level
// unit census with the verifier's own walk.
func deriveUnits(file *source.File, overlay map[string][]byte) ([]derivedUnit, error) {
	raw, overlaid := overlay[file.Path()]
	if !overlaid {
		var err error
		raw, err = os.ReadFile(file.Path())
		if err != nil {
			return nil, fmt.Errorf("%s: unreadable: %v", file.ID(), err)
		}
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, file.Path(), raw, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%s: unparsable: %v", file.ID(), err)
	}
	importsC := false
	for _, imported := range syntax.Imports {
		if imported.Path != nil && imported.Path.Value == `"C"` {
			importsC = true
		}
	}
	var out []derivedUnit
	record := func(node ast.Node, kind identity.UnitKind, cDependent bool) error {
		start := fset.PositionFor(node.Pos(), false).Offset
		end := fset.PositionFor(node.End(), false).Offset
		spanID, err := identity.NewSpanID(file.ID(), start, end)
		if err != nil {
			return err
		}
		unitID, err := identity.NewSourceUnitID(spanID, kind)
		if err != nil {
			return err
		}
		if end > len(raw) {
			return fmt.Errorf("%s: span exceeds bytes", unitID)
		}
		out = append(out, derivedUnit{
			id:         unitID,
			hash:       fmt.Sprintf("%x", sha256.Sum256(raw[start:end])),
			cDependent: cDependent,
		})
		return nil
	}
	for _, decl := range syntax.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			kind := identity.UnitFuncBody
			node := ast.Node(decl.Body)
			if decl.Body == nil {
				kind = identity.UnitBodylessDecl
				node = decl
			}
			if err := record(node, kind, importsC && usesC(decl)); err != nil {
				return nil, err
			}
		case *ast.GenDecl:
			if decl.Tok != token.VAR {
				continue
			}
			for _, spec := range decl.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok || len(value.Values) == 0 {
					continue
				}
				if err := record(value, identity.UnitVarInitializer, importsC && usesC(value)); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

// usesC is the verifier's own C-reference walk.
func usesC(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if selector, ok := n.(*ast.SelectorExpr); ok {
			if base, ok := selector.X.(*ast.Ident); ok && base.Name == "C" {
				found = true
			}
		}
		return !found
	})
	return found
}
