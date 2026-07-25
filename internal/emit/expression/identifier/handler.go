package identifier

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, source *ast.Ident) (tsgo.Expression, error) {
	object := context.TypesInfo().Uses[source]
	if object == nil {
		return nil, api.Unsupported(context, api.CategoryExpression, source)
	}
	switch object {
	case types.Universe.Lookup("false"):
		return context.Factory().FalseLiteral(), nil
	case types.Universe.Lookup("true"):
		return context.Factory().TrueLiteral(), nil
	}
	name, err := context.Names().Reference(object)
	if err != nil {
		return nil, err
	}
	return context.Factory().Identifier(name), nil
}
