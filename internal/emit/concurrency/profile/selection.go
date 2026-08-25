package profile

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func Admit(
	context api.Context,
	category api.Category,
	source ast.Node,
) error {
	if !context.ConcurrencySemantics().Valid() {
		return api.Unsupported(context, category, source)
	}
	return nil
}
