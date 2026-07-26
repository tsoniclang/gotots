package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	namedstructtype "github.com/tsoniclang/gotots/internal/emit/type/namedstruct"
	tupletype "github.com/tsoniclang/gotots/internal/emit/type/tuple"
)

func (e *emitter) Type(
	context api.Context,
	source ast.Expr,
) (api.TypeEmission, error) {
	if sourceType := context.TypesInfo().TypeOf(source); sourceType != nil {
		if _, ok := types.Unalias(sourceType).(*types.Named); ok {
			return namedstructtype.Emit(context, source, sourceType)
		}
	}
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
	if _, ok := types.Unalias(sourceType).(*types.Named); ok {
		return namedstructtype.Emit(context, source, sourceType)
	}
	return basictype.EmitRepresented(context, source, sourceType)
}
