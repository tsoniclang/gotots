package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	anonymousstruct "github.com/tsoniclang/gotots/internal/emit/type/anonymousstruct"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func anonymousStructZero(
	context api.Context,
	source ast.Node,
	structType *types.Struct,
) (api.ExpressionEmission, error) {
	return anonymousStructOperation(
		context,
		source,
		structType,
		api.AnonymousStructDemandZero,
		api.NamedStructOperationZero,
		nil,
	)
}

func anonymousStructCopy(
	context api.Context,
	source ast.Node,
	structType *types.Struct,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	target, err := anonymousStructOperation(
		context,
		source,
		structType,
		api.AnonymousStructDemandCopy,
		api.NamedStructOperationCopy,
		[]tsgo.Expression{value.Value()},
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		append(value.Before(), target.Before()...),
		target.Value(),
		api.CombineRequests(value.Requests(), target.Requests()),
	)
}

func anonymousStructEqual(
	context api.Context,
	source ast.Node,
	structType *types.Struct,
	left tsgo.Expression,
	right tsgo.Expression,
) (api.ExpressionEmission, error) {
	if !types.Comparable(structType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return anonymousStructOperation(
		context,
		source,
		structType,
		api.AnonymousStructDemandEqual,
		api.NamedStructOperationEqual,
		[]tsgo.Expression{left, right},
	)
}

func anonymousStructHash(
	context api.Context,
	source ast.Node,
	structType *types.Struct,
	value tsgo.Expression,
) (api.ExpressionEmission, error) {
	if !types.Comparable(structType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return anonymousStructOperation(
		context,
		source,
		structType,
		api.AnonymousStructDemandHash,
		api.NamedStructOperationHash,
		[]tsgo.Expression{value},
	)
}

func anonymousStructOperation(
	context api.Context,
	source ast.Node,
	structType *types.Struct,
	demand api.AnonymousStructDemand,
	operation api.NamedStructOperation,
	arguments []tsgo.Expression,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().AnonymousStruct(
		structType,
		demand,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	call, err := staticStructOperationCall(
		context,
		reference.Name(),
		operation,
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.DirectExpression(call, reference.Requests()...), nil
}

func isAnonymousStruct(sourceType types.Type) (*types.Struct, bool) {
	return anonymousstruct.Resolve(sourceType)
}
