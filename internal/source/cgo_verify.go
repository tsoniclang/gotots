package source

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
)

// verifyOriginGraph independently reconstructs the cgo origin evidence through
// a distinct path and exact-joins it against the producer's. Where the
// producer walks declarations forward and joins each element to its origin,
// the verifier walks the ORIGIN UNITS and, for each, independently locates the
// unique checked counterpart by reverse position lookup; it recomputes the
// synthetic-object set from the checker's Defs over non-original files; and it
// re-derives C-dependence. Any divergence in the counterpart set, the mapping
// spans, the synthetic identities, or the C-dependence classification fails
// closed with the exact one-sided difference.
func verifyOriginGraph(u *Universe, pkg *LoadedPackage, originals map[string]*LoadedFile, producer *originGraph) error {
	fail := func(reason string) error {
		return &LoadError{Dir: u.request.Dir, Reason: "cgo cross-check: " + reason}
	}
	// Independent counterpart index: every checked declaration element keyed
	// by its //line-adjusted origin position (file, line, column, kind).
	counterpartAt := map[originPosKey]ast.Node{}
	syntheticNames := map[string]bool{}
	syntheticObj := map[types.Object]bool{}
	for i := range pkg.checkedDecls {
		node := pkg.checkedDecls[i].node
		display := u.fset.Position(node.Pos())
		if _, isOriginal := originals[display.Filename]; !isOriginal {
			for _, decl := range syntheticDecls(node) {
				syntheticNames[decl.name] = true
			}
			collectDefinedObjects(node, pkg.typesInfo, syntheticObj)
			continue
		}
		for _, element := range declElements(node) {
			pos := u.fset.Position(element.joinNode.Pos())
			key := originPosKey{pos.Filename, pos.Line, pos.Column, element.kind}
			if _, dup := counterpartAt[key]; dup {
				return fail("two checked elements share origin position " + pos.String())
			}
			counterpartAt[key] = element.spanNode
		}
	}
	// Reverse join: for every origin unit, find its unique counterpart.
	independent := map[identity.SourceUnitID]ast.Node{}
	for _, file := range pkg.files {
		if !file.cgoOriginal {
			continue
		}
		for _, unit := range file.units {
			key := originPosKey{file.path, unit.span.Start.Line, unit.span.Start.Column, unit.id.Kind()}
			node, ok := counterpartAt[key]
			if !ok {
				// Column may be absent in //line evidence: fall back to the
				// unique same-file, same-line, same-kind element.
				node, ok = uniqueByLine(counterpartAt, file.path, unit.span.Start.Line, unit.id.Kind())
			}
			if !ok {
				return fail("no unique independent counterpart for unit " + unit.id.String())
			}
			independent[unit.id] = node
		}
	}
	// Exact-join the counterpart sets both ways.
	for id, counterpart := range producer.counterparts {
		other, ok := independent[id]
		if !ok {
			return fail("producer maps unit " + id.String() + " the independent join does not")
		}
		if other != counterpart.node {
			return fail("counterpart node diverges for unit " + id.String())
		}
	}
	for id := range independent {
		if _, ok := producer.counterparts[id]; !ok {
			return fail("independent join maps unit " + id.String() + " the producer does not")
		}
	}
	// Exact-join the synthetic identities.
	producerNames := map[string]bool{}
	for _, s := range producer.synthetics {
		producerNames[s.Name()] = true
	}
	for name := range syntheticNames {
		if !producerNames[name] {
			return fail("independent synthetic " + name + " absent from the producer set")
		}
	}
	for name := range producerNames {
		if !syntheticNames[name] {
			return fail("producer synthetic " + name + " absent from the independent set")
		}
	}
	// Independently re-derive C-dependence and exact-join against the producer
	// (which has not yet applied it — compare against a fresh derivation using
	// the producer's counterparts and the independent synthetic-object set).
	for _, file := range pkg.files {
		if !file.cgoOriginal {
			continue
		}
		for _, unit := range file.units {
			viaProducer := usesSyntheticObject(producer.counterparts[unit.id].node, pkg.typesInfo, producer.syntheticObj)
			viaIndependent := usesSyntheticObject(independent[unit.id], pkg.typesInfo, syntheticObj)
			if viaProducer != viaIndependent {
				return fail("C-dependence diverges for unit " + unit.id.String())
			}
		}
	}
	return nil
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
