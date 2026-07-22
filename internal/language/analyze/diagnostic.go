// Typed analysis diagnostics: immutable, constructor-validated records
// carrying all phase-available identity, kind, edge, role, and span evidence.
// Error strings are rendered only for human display and never parsed to
// select behavior.
package analyze

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// UnknownConstructError reports a concrete syntax node with no catalog Kind.
// Classification fails closed: an unrecognized form is a typed error, never a
// default classification. At the pure classification boundary only the Go
// type is available; the traversal boundary re-raises with file and span.
type UnknownConstructError struct {
	goType string
	file   identity.FileID
	span   Span
}

func newUnknownConstructError(goType string, file identity.FileID, span Span) *UnknownConstructError {
	return &UnknownConstructError{goType: goType, file: file, span: span}
}

// GoType is the concrete go/ast node type with no catalog identity.
func (e *UnknownConstructError) GoType() string { return e.goType }

// File is the owning file identity when the traversal boundary raised the
// error; zero at the pure classification boundary.
func (e *UnknownConstructError) File() identity.FileID { return e.file }

// Span is the physical span when available.
func (e *UnknownConstructError) Span() Span { return e.span }

func (e *UnknownConstructError) Error() string {
	return fmt.Sprintf("GOTOTS_UNKNOWN_CONSTRUCT: no catalog kind for go/ast node %s at %s#%d-%d",
		e.goType, e.file, e.span.Start.Offset, e.span.End.Offset)
}

// ConstructError reports a classified construct whose catalog disposition
// forbids inventory admission. The disposition is the catalog's own; the
// traversal hardcodes no per-kind rejection. Parent and edge carry the
// grammatical position when the rejection happened below the root.
type ConstructError struct {
	kind        catalog.Kind
	disposition catalog.Disposition
	file        identity.FileID
	span        Span
	parent      identity.OccurrenceID
	edge        catalog.Edge
}

func newConstructError(kind catalog.Kind, disposition catalog.Disposition,
	file identity.FileID, span Span, parent identity.OccurrenceID, edge catalog.Edge) *ConstructError {
	return &ConstructError{kind: kind, disposition: disposition, file: file, span: span, parent: parent, edge: edge}
}

// Kind is the rejected construct kind.
func (e *ConstructError) Kind() catalog.Kind { return e.kind }

// Disposition is the catalog disposition that forbade admission.
func (e *ConstructError) Disposition() catalog.Disposition { return e.disposition }

// File is the owning file identity.
func (e *ConstructError) File() identity.FileID { return e.file }

// Span is the physical span.
func (e *ConstructError) Span() Span { return e.span }

// Parent is the enclosing occurrence, zero at the root.
func (e *ConstructError) Parent() identity.OccurrenceID { return e.parent }

// Edge is the parent edge the rejection happened across, invalid at the root.
func (e *ConstructError) Edge() catalog.Edge { return e.edge }

func (e *ConstructError) Error() string {
	return fmt.Sprintf("GOTOTS_REJECTED_CONSTRUCT: %s construct %s at %s#%d-%d (edge %s) is not admissible",
		e.disposition, e.kind, e.file, e.span.Start.Offset, e.span.End.Offset, e.edge)
}

// TraversalDefectError is a compiler defect: the catalog-driven traversal met
// a structural impossibility (a cataloged edge missing from the toolchain
// struct, or a non-node value on a node edge). It indicates catalog/toolchain
// drift, never a user error.
type TraversalDefectError struct {
	edge   catalog.Edge
	goType string
	file   identity.FileID
	reason string
}

func newTraversalDefectError(edge catalog.Edge, goType string, file identity.FileID, reason string) *TraversalDefectError {
	return &TraversalDefectError{edge: edge, goType: goType, file: file, reason: reason}
}

// Edge is the edge being traversed.
func (e *TraversalDefectError) Edge() catalog.Edge { return e.edge }

// GoType is the concrete parent node type.
func (e *TraversalDefectError) GoType() string { return e.goType }

// File is the owning file identity.
func (e *TraversalDefectError) File() identity.FileID { return e.file }

func (e *TraversalDefectError) Error() string {
	return fmt.Sprintf("GOTOTS_COMPILER_DEFECT: traversing %s on %s in %s: %s",
		e.edge, e.goType, e.file, e.reason)
}

// UnknownDirectiveError reports a go:* comment directive outside the closed
// directive catalog. Unknown go directives can carry semantics and fail
// inventory closed.
type UnknownDirectiveError struct {
	tool string
	name string
	file identity.FileID
	span Span
}

func newUnknownDirectiveError(tool, name string, file identity.FileID, span Span) *UnknownDirectiveError {
	return &UnknownDirectiveError{tool: tool, name: name, file: file, span: span}
}

// Tool is the directive tool ("go").
func (e *UnknownDirectiveError) Tool() string { return e.tool }

// Name is the unknown directive name.
func (e *UnknownDirectiveError) Name() string { return e.name }

// File is the owning file identity.
func (e *UnknownDirectiveError) File() identity.FileID { return e.file }

// Span is the directive's physical span.
func (e *UnknownDirectiveError) Span() Span { return e.span }

func (e *UnknownDirectiveError) Error() string {
	return fmt.Sprintf("GOTOTS_UNKNOWN_DIRECTIVE: //%s:%s at %s#%d-%d is not in the directive catalog",
		e.tool, e.name, e.file, e.span.Start.Offset, e.span.End.Offset)
}

// ResolutionError is a typed failure to resolve a semantic variant or token
// binding for one occurrence; unresolvable variant-bearing occurrences fail
// closed.
type ResolutionError struct {
	kind   catalog.Kind
	file   identity.FileID
	span   Span
	reason string
}

func newResolutionError(kind catalog.Kind, file identity.FileID, span Span, reason string) *ResolutionError {
	return &ResolutionError{kind: kind, file: file, span: span, reason: reason}
}

// Kind is the occurrence kind that failed to resolve.
func (e *ResolutionError) Kind() catalog.Kind { return e.kind }

// File is the owning file identity.
func (e *ResolutionError) File() identity.FileID { return e.file }

// Span is the occurrence's physical span.
func (e *ResolutionError) Span() Span { return e.span }

func (e *ResolutionError) Error() string {
	return fmt.Sprintf("GOTOTS_UNRESOLVED_OCCURRENCE: %s at %s#%d-%d: %s",
		e.kind, e.file, e.span.Start.Offset, e.span.End.Offset, e.reason)
}
