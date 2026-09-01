package sourcefact

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type Error struct {
	Symbol api.RuntimeSymbol
}

type AnnotationError struct {
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("build Go source fact declaration: invalid runtime symbol %d", e.Symbol)
}

func (e *AnnotationError) Error() string {
	return "build Go source fact annotation: " + e.Reason
}
