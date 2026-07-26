package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	tupletype "github.com/tsoniclang/gotots/internal/emit/type/tuple"
)

func (e *emitter) Type(
	context api.Context,
	source ast.Expr,
) (api.TypeEmission, error) {
	return basictype.Emit(context, source)
}

func (e *emitter) RepresentedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	if tuple, ok := types.Unalias(sourceType).(*types.Tuple); ok {
		return tupletype.Emit(context, e, source, tuple)
	}
	return basictype.EmitRepresented(context, source, sourceType)
}
