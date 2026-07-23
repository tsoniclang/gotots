// Package frontend is the sole transient Go-checker-to-semantic materializer.
package frontend

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

// Error is a typed Stage-2 materialization failure.
type Error struct {
	Package    identity.PackageID
	Definition identity.DefinitionID
	Occurrence identity.OccurrenceID
	Kind       catalog.Kind
	Reason     string
}

func (err *Error) Error() string {
	return fmt.Sprintf(
		"GOTOTS_FRONTEND: package=%s definition=%s occurrence=%s kind=%s: %s",
		err.Package,
		err.Definition,
		err.Occurrence,
		err.Kind,
		err.Reason,
	)
}
