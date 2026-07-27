package api

import (
	"go/ast"
	"go/types"
)

type AddressableStorage interface {
	Name(Context, *types.Var) (string, bool)
	Read(Context, *types.Var) (ExpressionEmission, bool)
	StoreTarget(Context, *types.Var) (StoreTargetEmission, bool, error)
	Cell(
		Context,
		ChildEmitter,
		ast.Node,
		types.Type,
		ExpressionEmission,
	) (ExpressionEmission, error)
	Requirement(Context, *types.Var) (RootRequest, error)
}
