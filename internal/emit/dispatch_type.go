package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
)

func (e *Emitter) Type(
	context api.Context,
	source ast.Expr,
) (api.TypeEmission, error) {
	return basictype.Emit(context, source)
}

func (e *Emitter) RepresentedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, error) {
	return basictype.EmitRepresented(context, source, sourceType)
}
