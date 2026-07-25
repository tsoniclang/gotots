package basic

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(context api.Context, source ast.Expr) (tsgo.TypeNode, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return nil, api.Unsupported(context, api.CategoryType, source)
	}
	basic, ok := types.Unalias(sourceType).(*types.Basic)
	if !ok || basic.Kind() != types.Int {
		return nil, api.Unsupported(context, api.CategoryType, source)
	}
	var targetName string
	switch context.TypesSizes().Sizeof(types.Typ[types.Int]) {
	case 4:
		targetName = "int32"
	case 8:
		targetName = "int64"
	default:
		return nil, api.Unsupported(context, api.CategoryType, source)
	}
	localName, err := context.Placement().TypeImport("@tsonic/core/types.js", targetName)
	if err != nil {
		return nil, err
	}
	return context.Factory().TypeReferenceNode(context.Factory().Identifier(localName), nil), nil
}
