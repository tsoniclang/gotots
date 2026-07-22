// Typed analysis diagnostics. Every rejection carries its construct kind,
// disposition, file identity, and span; error strings are rendered only for
// human display and are never parsed to select behavior.
package analyze

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// UnknownConstructError reports a concrete syntax node with no catalog Kind.
// Classification fails closed: an unrecognized form is a typed error, never a
// default or best-effort classification. File and Span are filled by the
// visitor, the boundary that owns that evidence.
type UnknownConstructError struct {
	// GoType is the concrete go/ast node type that has no catalog identity.
	GoType string
	File   identity.FileID
	Span   Span
}

func (e *UnknownConstructError) Error() string {
	return fmt.Sprintf("GOTOTS_UNKNOWN_CONSTRUCT: no catalog kind for go/ast node %s at %s#%d-%d",
		e.GoType, e.File, e.Span.Start.Offset, e.Span.End.Offset)
}

// ConstructError reports a classified construct whose catalog disposition
// forbids inventory admission (deprecated and recovery forms). The disposition
// is the catalog's; the visitor never hardcodes a per-kind rejection.
type ConstructError struct {
	Kind        catalog.Kind
	Disposition catalog.Disposition
	File        identity.FileID
	Span        Span
}

func (e *ConstructError) Error() string {
	return fmt.Sprintf("GOTOTS_REJECTED_CONSTRUCT: %s construct %s at %s#%d-%d is not admissible",
		e.Disposition, e.Kind, e.File, e.Span.Start.Offset, e.Span.End.Offset)
}

// UnvisitedConstructError is a compiler defect: a classified, admissible
// construct reached the traversal without a child-visit arm. It indicates the
// visitor's coverage lags the catalog, never a user error.
type UnvisitedConstructError struct {
	GoType string
	File   identity.FileID
}

func (e *UnvisitedConstructError) Error() string {
	return fmt.Sprintf("GOTOTS_COMPILER_DEFECT: no child-visit arm for %s in %s", e.GoType, e.File)
}
