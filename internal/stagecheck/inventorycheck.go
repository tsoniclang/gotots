package stagecheck

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/analyze"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// kindByName joins toolchain concrete type names to pinned catalog kinds. The
// catalog's pinned vocabulary is the shared contract; the verifier does not
// call the producer's classifier binding.
var kindByName = func() map[string]catalog.Kind {
	m := map[string]catalog.Kind{}
	for _, kind := range catalog.All() {
		m[kind.Name()] = kind
	}
	return m
}()

// VerifySyntaxInventory independently reconciles the inventory artifact
// against the toolchain's own traversal: for every file it walks the syntax
// with ast.Inspect (a producer-independent walker), reconstructs the expected
// occurrence identity at every preorder position, and exact-joins node
// identity, order, parent edges, and canonical IDs. Dropped nodes, orphan
// occurrences, duplicate identities, and downstream reinterpretation of a
// node's kind all fail.
func VerifySyntaxInventory(ws *source.Workspace, inv *analyze.WorkspaceInventory) error {
	fail := func(reason string) error {
		return &VerificationError{Stage: "syntax-inventory", Reason: reason}
	}
	files := map[string]*source.File{}
	for _, pkg := range ws.Packages() {
		for _, file := range pkg.Files() {
			if file.Syntax() != nil {
				files[file.ID().String()] = file
			}
		}
	}
	verified := 0
	for _, pkg := range inv.Packages() {
		for _, fileInv := range pkg.Files() {
			file, ok := files[fileInv.File().String()]
			if !ok {
				return fail("orphan inventory: no workspace file " + fileInv.File().String())
			}
			if err := verifyFileInventory(file, fileInv); err != nil {
				return err
			}
			verified++
		}
	}
	if verified != len(files) {
		return fail(fmt.Sprintf("dropped input: workspace has %d files, inventory covers %d", len(files), verified))
	}
	return nil
}

func verifyFileInventory(file *source.File, inv *analyze.FileInventory) error {
	fail := func(reason string) error {
		return &VerificationError{Stage: "syntax-inventory", Reason: inv.File().String() + ": " + reason}
	}
	type expectation struct {
		typeName string
		start    int
		end      int
		parent   int
	}
	var walk []expectation
	var stack []int
	ast.Inspect(file.Syntax(), func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		parent := -1
		if len(stack) > 0 {
			parent = stack[len(stack)-1]
		}
		walk = append(walk, expectation{
			typeName: strings.TrimPrefix(fmt.Sprintf("%T", n), "*ast."),
			start:    file.Fset().PositionFor(n.Pos(), false).Offset,
			end:      file.Fset().PositionFor(n.End(), false).Offset,
			parent:   parent,
		})
		stack = append(stack, len(walk)-1)
		return true
	})
	occurrences := inv.Occurrences()
	if len(occurrences) != len(walk) {
		return fail(fmt.Sprintf("inventory has %d occurrences, independent walk has %d", len(occurrences), len(walk)))
	}
	seen := map[identity.OccurrenceID]bool{}
	for i, occurrence := range occurrences {
		expected := walk[i]
		kind, known := kindByName[expected.typeName]
		if !known {
			return fail("independent walk met unknown node type " + expected.typeName)
		}
		if occurrence.Kind() != kind {
			return fail(fmt.Sprintf("preorder %d reinterprets %s as %s", i, expected.typeName, occurrence.Kind()))
		}
		if occurrence.Span().Start.Offset != expected.start || occurrence.Span().End.Offset != expected.end {
			return fail(fmt.Sprintf("preorder %d (%s) span %d-%d, walk has %d-%d", i, kind,
				occurrence.Span().Start.Offset, occurrence.Span().End.Offset, expected.start, expected.end))
		}
		spanID, err := identity.NewSpanID(inv.File(), expected.start, expected.end)
		if err != nil {
			return fail("independent span identity: " + err.Error())
		}
		expectedID, err := identity.NewOccurrenceID(spanID, uint16(kind))
		if err != nil {
			return fail("independent occurrence identity: " + err.Error())
		}
		if occurrence.ID() != expectedID {
			return fail(fmt.Sprintf("preorder %d identity %s, independently derived %s", i, occurrence.ID(), expectedID))
		}
		if seen[occurrence.ID()] {
			return fail("duplicate occurrence identity " + occurrence.ID().String())
		}
		seen[occurrence.ID()] = true
		var expectedParent identity.OccurrenceID
		if expected.parent >= 0 {
			expectedParent = occurrences[expected.parent].ID()
		}
		if occurrence.Parent() != expectedParent {
			return fail(fmt.Sprintf("preorder %d parent %s, walk has %s", i, occurrence.Parent(), expectedParent))
		}
		if catalog.VariantBearing(occurrence.Kind()) && !occurrence.Variant().Valid() {
			return fail(fmt.Sprintf("preorder %d (%s) variant unresolved", i, kind))
		}
	}
	return nil
}
