package array

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (array RuntimeArray) MemoryStorageType(context api.Context, source ast.Node) (api.TypeEmission, error) {
	element, err := context.Values().MemoryStorageType(context.WithRole(api.RoleStorageType), source, array.ElementType())
	if err != nil {
		return api.TypeEmission{}, err
	}
	reference, err := context.Names().TsonicCore(tsoniccore.SymbolFixedArray)
	if err != nil {
		return api.TypeEmission{}, err
	}
	return api.DirectType(context.Factory().TypeReferenceNode(reference.EntityName(context.Factory()), []tsgo.TypeNode{
		element.Value(), context.Factory().LiteralTypeNode(array.lengthLiteral(context)),
	}), api.CombineRequests(reference.Requests(), element.Requests())...), nil
}

func (array RuntimeArray) Location(context api.Context, children api.ChildEmitter, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	stored, err := array.storage(context, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	location, requests, err := array.runtimeOperation(context, children, api.RuntimeArrayLocation, stored.Value())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(stored.Before(), location, api.CombineRequests(stored.Requests(), requests))
}

func (array RuntimeArray) FromRegion(context api.Context, children api.ChildEmitter, value api.ExpressionEmission) (api.ExpressionEmission, error) {
	result, requests, err := array.runtimeOperation(context, children, api.RuntimeArrayFromRegion, value.Value(), array.lengthLiteral(context))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	restored, err := api.NewExpressionEmission(value.Before(), result, api.CombineRequests(value.Requests(), requests))
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return array.wrap(context, restored)
}

func (array RuntimeArray) PhysicalZero(context api.Context, source ast.Node) (api.ExpressionEmission, error) {
	if context.TypesSizes().Sizeof(array.source) != 0 {
		return api.ExpressionEmission{}, api.Unsupported(context, api.CategoryExpression, source)
	}
	storage, err := array.MemoryStorageType(context, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return pointermarker.Operation(context, tsoniccore.SymbolDefaultValue, []api.TypeEmission{storage}, nil)
}
