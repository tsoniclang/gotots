package source

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
)

// verifyOriginGraph independently reconstructs the cgo origin evidence through
// a SEPARATELY IMPLEMENTED extractor and exact-joins it against the producer's.
// It shares none of the producer's critical enumeration or classification
// helpers: it walks each checked declaration with its own ast.Inspect pass to
// collect unit-bearing nodes, classifies synthetics with its own logic, and
// joins the counterpart set and the complete synthetic identity (package, name,
// and role — never the name alone). Semantic C-dependence is proven separately
// through selected-Go fixtures and mutations, not re-derived here.
func verifyOriginGraph(u *Universe, pkg *LoadedPackage, originals map[string]*LoadedFile, producer *originGraph) error {
	fail := func(reason string) error {
		return &LoadError{Dir: u.request.Dir, Reason: "cgo cross-check: " + reason}
	}

	// Independent counterpart index and synthetic identity set.
	counterpartAt := map[originPosKey]ast.Node{}
	independentSynthetics := map[string]bool{} // pkg|name|role
	for i := range pkg.checkedDecls {
		root := pkg.checkedDecls[i].node
		display := u.fset.Position(root.Pos())
		if _, isOriginal := originals[display.Filename]; !isOriginal {
			for key := range extractSyntheticKeys(pkg.id, root) {
				independentSynthetics[key] = true
			}
			continue
		}
		// Own enumeration: collect unit-bearing (joinNode, spanNode, kind)
		// triples by a direct ast.Inspect walk, not the producer's declElements.
		var walkErr error
		ast.Inspect(root, func(n ast.Node) bool {
			if walkErr != nil {
				return false
			}
			switch node := n.(type) {
			case *ast.FuncDecl:
				kind, join := identity.UnitFuncBody, ast.Node(node.Body)
				if node.Body == nil {
					kind, join = identity.UnitBodylessDecl, node
				}
				walkErr = indexCounterpart(u, counterpartAt, join, node, kind, fail)
			case *ast.FuncLit:
				walkErr = indexCounterpart(u, counterpartAt, node, node, identity.UnitFuncLitBody, fail)
			case *ast.ValueSpec:
				if len(node.Values) > 0 {
					walkErr = indexCounterpart(u, counterpartAt, node, node, identity.UnitVarInitializer, fail)
				}
			}
			return walkErr == nil
		})
		if walkErr != nil {
			return walkErr
		}
	}

	// Reverse join: for every origin unit, locate its unique counterpart.
	independent := map[identity.SourceUnitID]ast.Node{}
	for _, file := range pkg.files {
		if !file.cgoOriginal {
			continue
		}
		for _, unit := range file.units {
			key := originPosKey{file.path, unit.span.Start.Line, unit.span.Start.Column, unit.id.Kind()}
			node, ok := counterpartAt[key]
			if !ok {
				node, ok = uniqueByLine(counterpartAt, file.path, unit.span.Start.Line, unit.id.Kind())
			}
			if !ok {
				return fail("no unique independent counterpart for unit " + unit.id.String())
			}
			independent[unit.id] = node
		}
	}

	// Exact-join counterparts both ways.
	for id, counterpart := range producer.counterparts {
		other, ok := independent[id]
		if !ok {
			return fail("producer maps unit " + id.String() + " the independent extractor does not")
		}
		if other != counterpart.node {
			return fail("counterpart node diverges for unit " + id.String())
		}
	}
	for id := range independent {
		if _, ok := producer.counterparts[id]; !ok {
			return fail("independent extractor maps unit " + id.String() + " the producer does not")
		}
	}

	// Exact-join complete synthetic identities (package, name, role).
	producerSynthetics := map[string]bool{}
	for _, s := range producer.synthetics {
		producerSynthetics[syntheticKey(s.Pkg(), s.Name(), s.Role())] = true
	}
	for key := range independentSynthetics {
		if !producerSynthetics[key] {
			return fail("independent synthetic identity " + key + " absent from the producer set")
		}
	}
	for key := range producerSynthetics {
		if !independentSynthetics[key] {
			return fail("producer synthetic identity " + key + " absent from the independent set")
		}
	}
	return nil
}

// indexCounterpart records one unit-bearing element in the counterpart index by
// its //line-adjusted origin position, failing on a shared position.
func indexCounterpart(u *Universe, index map[originPosKey]ast.Node, joinNode, spanNode ast.Node, kind identity.UnitKind, fail func(string) error) error {
	pos := u.fset.Position(joinNode.Pos())
	key := originPosKey{pos.Filename, pos.Line, pos.Column, kind}
	if _, dup := index[key]; dup {
		return fail("two checked elements share origin position " + pos.String())
	}
	index[key] = spanNode
	return nil
}

// extractSyntheticKeys independently classifies one generated declaration's
// synthetic identities as package|name|role keys, with its own role logic.
func extractSyntheticKeys(pkg identity.PackageID, node ast.Node) map[string]bool {
	out := map[string]bool{}
	switch n := node.(type) {
	case *ast.FuncDecl:
		if n.Name != nil && n.Name.Name != "" {
			name := n.Name.Name
			if n.Recv != nil && len(n.Recv.List) > 0 {
				name = "(" + recvDisplay(n.Recv.List[0]) + ")." + n.Name.Name
			}
			out[syntheticKey(pkg, name, SyntheticAdapter)] = true
		}
	case *ast.GenDecl:
		switch n.Tok {
		case token.TYPE:
			for _, spec := range n.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name != nil {
					out[syntheticKey(pkg, ts.Name.Name, SyntheticTypeDecl)] = true
				}
			}
		case token.VAR, token.CONST:
			for _, spec := range n.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, name := range vs.Names {
						if name.Name != "" && name.Name != "_" {
							out[syntheticKey(pkg, name.Name, SyntheticData)] = true
						}
					}
				}
			}
		}
	}
	return out
}

// syntheticKey is the canonical package|name|role join key.
func syntheticKey(pkg identity.PackageID, name string, role SyntheticRole) string {
	return pkg.String() + "|" + name + "|" + role.String()
}

// originPosKey keys a checked element by its //line-adjusted origin position
// and unit kind.
type originPosKey struct {
	file   string
	line   int
	column int
	kind   identity.UnitKind
}

// uniqueByLine returns the single counterpart element on one file+line of a
// given kind, if exactly one exists.
func uniqueByLine(index map[originPosKey]ast.Node, file string, line int, kind identity.UnitKind) (ast.Node, bool) {
	var found ast.Node
	count := 0
	for key, node := range index {
		if key.file == file && key.line == line && key.kind == kind {
			found = node
			count++
		}
	}
	if count == 1 {
		return found, true
	}
	return nil, false
}
