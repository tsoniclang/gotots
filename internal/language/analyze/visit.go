package analyze

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// builder accumulates one file's occurrence records during the catalog-driven
// traversal. The edge catalog is the single owner of child-edge knowledge: the
// traversal reflects over exactly the cataloged edges in pinned order and
// holds no private edge table. nodes/parentIdx run parallel to occurrences so
// the resolvers can consult typed context without re-walking.
type builder struct {
	fset        *token.FileSet
	file        identity.FileID
	occurrences []Occurrence
	nodes       []ast.Node
	parentIdx   []int
	index       map[ast.Node]int
	// boundaries are nested-unit root nodes that bound this region: the
	// traversal records nothing for them and does not descend, so a nested
	// unit's interior is unreachable through its parent's inventory.
	boundaries map[ast.Node]bool
}

// visit records n (hanging from parentIdx across edge) and descends into its
// cataloged child edges. The catalog disposition is the single admission
// policy: the switch is exhaustive and holds no per-kind special case.
func (b *builder) visit(n ast.Node, parentIdx int, edge catalog.Edge) error {
	if b.boundaries[n] {
		// A nested unit's root: its region is inventoried independently, so
		// this parent records neither the boundary node nor its interior.
		return nil
	}
	kind, err := Classify(n)
	if err != nil {
		if unknown, ok := err.(*UnknownConstructError); ok {
			return newUnknownConstructError(unknown.GoType(), b.file, b.physicalSpan(n))
		}
		return err
	}
	span := b.physicalSpan(n)
	parentID := identity.OccurrenceID{}
	if parentIdx >= 0 {
		parentID = b.occurrences[parentIdx].id
	}
	switch disposition := kind.Disposition(); disposition {
	case catalog.DispositionActive:
		// admissible
	case catalog.DispositionDeprecated, catalog.DispositionRecovery:
		return newConstructError(kind, disposition, b.file, span, parentID, edge)
	default:
		return newConstructError(kind, catalog.DispositionInvalid, b.file, span, parentID, edge)
	}
	spanID, err := identity.NewSpanID(b.file, span.Start.Offset, span.End.Offset)
	if err != nil {
		return err
	}
	id, err := identity.NewOccurrenceID(spanID, uint16(kind))
	if err != nil {
		return err
	}
	tokenKind, err := b.tokenEvidence(n, kind, span)
	if err != nil {
		return err
	}
	b.occurrences = append(b.occurrences, Occurrence{
		id: id, kind: kind, parent: parentID, edge: edge,
		span: span, display: b.displaySpan(n), token: tokenKind,
	})
	b.nodes = append(b.nodes, n)
	b.parentIdx = append(b.parentIdx, parentIdx)
	selfIdx := len(b.occurrences) - 1
	if b.index == nil {
		b.index = map[ast.Node]int{}
	}
	b.index[n] = selfIdx

	value := reflect.ValueOf(n).Elem()
	for _, childEdge := range catalog.EdgesOf(kind) {
		field := value.FieldByName(childEdge.Field())
		if !field.IsValid() {
			return newTraversalDefectError(childEdge, fmt.Sprintf("%T", n), b.file, "cataloged field missing from toolchain struct")
		}
		if childEdge.IsList() {
			for i := 0; i < field.Len(); i++ {
				if err := b.visitValue(field.Index(i), selfIdx, childEdge, n); err != nil {
					return err
				}
			}
			continue
		}
		if err := b.visitValue(field, selfIdx, childEdge, n); err != nil {
			return err
		}
	}
	return nil
}

// visitValue descends into one child slot; a nil pointer or interface is a
// genuinely absent optional edge.
func (b *builder) visitValue(v reflect.Value, parentIdx int, edge catalog.Edge, parent ast.Node) error {
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return nil
		}
	default:
		return newTraversalDefectError(edge, fmt.Sprintf("%T", parent), b.file, "cataloged field is not a node slot")
	}
	child, ok := v.Interface().(ast.Node)
	if !ok {
		return newTraversalDefectError(edge, fmt.Sprintf("%T", parent), b.file, "cataloged field value is not an ast.Node")
	}
	return b.visit(child, parentIdx, edge)
}

// tokenEvidence binds the lexical token evidence of token-bearing kinds
// through the token catalog; an unmapped toolchain token fails closed.
func (b *builder) tokenEvidence(n ast.Node, kind catalog.Kind, span Span) (catalog.TokenKind, error) {
	var tok token.Token
	switch n := n.(type) {
	case *ast.BinaryExpr:
		tok = n.Op
	case *ast.UnaryExpr:
		tok = n.Op
	case *ast.AssignStmt:
		tok = n.Tok
	case *ast.IncDecStmt:
		tok = n.Tok
	case *ast.BranchStmt:
		tok = n.Tok
	case *ast.GenDecl:
		tok = n.Tok
	case *ast.BasicLit:
		tok = n.Kind
	default:
		return 0, nil
	}
	bound := catalog.TokenBySpelling(tok.String())
	if !bound.Valid() {
		return 0, newResolutionError(kind, b.file, span, "toolchain token "+tok.String()+" is not in the token catalog")
	}
	return bound, nil
}

// physicalSpan measures n with //line directives ignored; these offsets enter
// the canonical identity.
func (b *builder) physicalSpan(n ast.Node) Span {
	start := b.fset.PositionFor(n.Pos(), false)
	end := b.fset.PositionFor(n.End(), false)
	return Span{
		Start: Position{Line: start.Line, Column: start.Column, Offset: start.Offset},
		End:   Position{Line: end.Line, Column: end.Column, Offset: end.Offset},
	}
}

// displaySpan measures n with //line directives applied, for diagnostics only.
func (b *builder) displaySpan(n ast.Node) DisplaySpan {
	start := b.fset.Position(n.Pos())
	end := b.fset.Position(n.End())
	return DisplaySpan{
		Filename: start.Filename,
		Start:    Position{Line: start.Line, Column: start.Column, Offset: start.Offset},
		End:      Position{Line: end.Line, Column: end.Column, Offset: end.Offset},
	}
}

// parentNode is the enclosing syntax node of the occurrence at index i, or nil
// at the root.
func (b *builder) parentNode(i int) ast.Node {
	p := b.parentIdx[i]
	if p < 0 {
		return nil
	}
	return b.nodes[p]
}

// enclosingFunc walks up from index i to the nearest function declaration or
// literal and returns its type, or nil when outside any function.
func (b *builder) enclosingFunc(i int) *ast.FuncType {
	for p := b.parentIdx[i]; p >= 0; p = b.parentIdx[p] {
		switch fn := b.nodes[p].(type) {
		case *ast.FuncDecl:
			return fn.Type
		case *ast.FuncLit:
			return fn.Type
		}
	}
	return nil
}
