package identifier

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, source *ast.Ident) (tsgo.Identifier, error) {
	object := context.TypesInfo().Uses[source]
	if object == nil {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	name, err := context.Names().Reference(object)
	if err != nil {
		return nil, err
	}
	return context.Factory().Identifier(name), nil
}
