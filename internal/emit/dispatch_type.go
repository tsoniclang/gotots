package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *Emitter) Type(context api.Context, source ast.Expr) (tsgo.TypeNode, error) {
	return basictype.Emit(context, source)
}

func (e *Emitter) RepresentedType(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, error) {
	return basictype.EmitRepresented(context, source, sourceType)
}
