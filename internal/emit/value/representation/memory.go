package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/value/representation/memory"
)

func (owner Owner) MemoryStorageType(context api.Context, source ast.Node, sourceType types.Type) (api.TypeEmission, error) {
	return memory.NewOwner(context, owner.children).MemoryStorageType(context, source, sourceType)
}

func (owner Owner) ToMemoryStorage(context api.Context, source ast.Node, sourceType types.Type, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	return memory.NewOwner(context, owner.children).ToMemoryStorage(context, source, sourceType, value)
}

func (owner Owner) FromMemoryStorage(context api.Context, source ast.Node, sourceType types.Type, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	return memory.NewOwner(context, owner.children).FromMemoryStorage(context, source, sourceType, value)
}
