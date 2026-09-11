package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

func (staticSpecializationValues) MemoryStorageType(api.Context, ast.Node, types.Type) (api.TypeEmission, error) {
	panic("map specialization does not request physical memory")
}

func (staticSpecializationValues) ToMemoryStorage(api.Context, ast.Node, types.Type, api.ExpressionEmission) (api.ExpressionEmission, error) {
	panic("map specialization does not request physical memory")
}

func (staticSpecializationValues) FromMemoryStorage(api.Context, ast.Node, types.Type, api.ExpressionEmission) (api.ExpressionEmission, error) {
	panic("map specialization does not request physical memory")
}

func (staticSpecializationValues) ProjectMemoryPointer(api.Context, ast.Node, types.Type, api.ExpressionEmission) (api.ExpressionEmission, error) {
	panic("map specialization does not request physical memory")
}
