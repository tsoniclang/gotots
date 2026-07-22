package analyze

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// builder accumulates one file's occurrence records during the catalog-driven
// traversal. The edge catalog is the single owner of child-edge knowledge: the
// traversal reflects over exactly the cataloged edges in pinned order and
// holds no private edge table. nodes/parentIdx run parallel to occurrences so
// the resolvers can consult typed context without re-walking.
type builder struct {
	fset        *token.FileSet
	file        identity.FileID
	owner       RegionOwner
	info        *source.TypeInfoView
	occurrences []Occurrence
	nodes       []ast.Node
	contexts    []visitContext // parent-assigned context, parallel to occurrences
	index       map[ast.Node]int
	// boundaries maps each nested-unit root node to its unit identity. At a
	// boundary the traversal records a typed implementation reference and does
	// not descend, so the child body is absent from this region — the parent
	// keeps the operation as a reference, never a raw child pointer.
	boundaries map[ast.Node]identity.SourceUnitID
	references []ImplementationRef
}

// visitContext is the grammatical context a parent visitor assigns to a child
// during descent. The child records the assigned context; it never inspects
// its parent afterward. All context is parent-directed: one owner supplies the
// role.
type visitContext struct {
	commaOk       bool          // the child is the single value of a two-target binding
	compositeType types.Type    // the enclosing composite literal's aggregate shape
	signature     *ast.FuncType // the enclosing function signature (for return forms)
	varDecl       bool          // the child spec belongs to a `var` declaration
}

// visit records n (hanging from parentIdx across edge, with parent-assigned
// context ctx) and descends into its cataloged child edges, assigning each
// child its context. At a nested-unit boundary it records a typed
// implementation reference and does not descend, so the child body is absent
// from this region.
func (b *builder) visit(n ast.Node, parentIdx int, edge catalog.Edge, ctx visitContext) error {
	if parentIdx >= 0 {
		if child, ok := b.boundaries[n]; ok {
			contract, err := ContractForKind(child.Kind())
			if err != nil {
				return newResolutionError(0, b.file, b.physicalSpan(n), err.Error())
			}
			b.references = append(b.references, ImplementationRef{
				parent:    b.owner,
				parentOcc: b.occurrences[parentIdx].id,
				edge:      edge,
				child:     SourceUnitRef(child),
				contract:  contract,
				anchor:    child.Span(),
				ordinal:   len(b.references),
			})
			return nil
		}
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
	b.contexts = append(b.contexts, ctx)
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
				if err := b.visitValue(field.Index(i), selfIdx, childEdge, n, kind, ctx, i, field.Len()); err != nil {
					return err
				}
			}
			continue
		}
		if err := b.visitValue(field, selfIdx, childEdge, n, kind, ctx, 0, 1); err != nil {
			return err
		}
	}
	return nil
}

// visitValue descends into one child slot; a nil pointer or interface is a
// genuinely absent optional edge. The parent assigns the child's context here.
func (b *builder) visitValue(v reflect.Value, parentIdx int, edge catalog.Edge, parent ast.Node, parentKind catalog.Kind, parentCtx visitContext, index, count int) error {
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
	return b.visit(child, parentIdx, edge, b.childContext(parent, parentKind, edge, child, parentCtx, index, count))
}

// childContext is the parent-directed context assignment: the parent visitor
// decides its child's grammatical role from its own structure. The child never
// inspects the parent afterward.
func (b *builder) childContext(parent ast.Node, parentKind catalog.Kind, edge catalog.Edge, child ast.Node, ctx visitContext, index, count int) visitContext {
	// Signature and composite-type context flow down unchanged unless a new
	// enclosing owner overrides them; comma-ok and var-decl are single-edge.
	out := visitContext{signature: ctx.signature, compositeType: ctx.compositeType}
	switch p := parent.(type) {
	case *ast.FuncLit:
		if edge.Field() == "Body" {
			out.signature = p.Type
		}
	case *ast.AssignStmt:
		if edge.Field() == "Rhs" && len(p.Lhs) == 2 && len(p.Rhs) == 1 {
			out.commaOk = true
		}
	case *ast.ValueSpec:
		if edge.Field() == "Values" && len(p.Names) == 2 && len(p.Values) == 1 {
			out.commaOk = true
		}
	case *ast.GenDecl:
		if edge.Field() == "Specs" && p.Tok == token.VAR {
			out.varDecl = true
		}
	case *ast.CompositeLit:
		if edge.Field() == "Elts" {
			if tv, ok := b.info.TypeOf(p); ok {
				out.compositeType = aggregateShape(tv.Type)
			}
		}
	}
	return out
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
