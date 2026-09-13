package binary

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (unusedValues) Pointee(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) ProjectStoragePointer(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) MemoryStorageType(api.Context, ast.Node, types.Type) (api.TypeEmission, error) {
	panic("unused")
}

func (unusedValues) ToMemoryStorage(api.Context, ast.Node, types.Type, api.ExpressionEmission) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) FromMemoryStorage(api.Context, ast.Node, types.Type, api.ExpressionEmission) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) ProjectMemoryPointer(api.Context, ast.Node, types.Type, api.ExpressionEmission) (api.ExpressionEmission, error) {
	panic("unused")
}
