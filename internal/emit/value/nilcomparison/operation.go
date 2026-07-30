package nilcomparison

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	interfacetype "github.com/tsoniclang/gotots/internal/emit/type/interfacevalue"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Apply(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if sourceType == nil {
		return api.ExpressionEmission{}, false, nil
	}
	if _, ok := interfacetype.Resolve(sourceType); ok {
		return direct(context, value)
	}
	if model, ok := maprepresentation.Source(context, sourceType); ok {
		if !model.Nominal() {
			name, nameErr := mapruntime.Name(mapruntime.MemberIsNil)
			return method(
				context,
				value,
				name,
				nameErr,
			)
		}
	}
	if defined, ok := definedtype.Resolve(sourceType); ok {
		if !defined.NilCapable() {
			return api.ExpressionEmission{}, false, nil
		}
		projected, err := defined.Project(context, value)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		return Apply(context, source, defined.Underlying(), projected)
	}
	if _, _, ok := slicevalue.Resolve(sourceType); ok {
		return method(
			context,
			value,
			runtimeslice.MemberName(runtimeslice.MemberIsNil),
			nil,
		)
	}
	switch types.Unalias(sourceType).(type) {
	case *types.Pointer, *types.Chan:
		return direct(context, value)
	}
	if _, ok := callable.Signature(sourceType); ok {
		return direct(context, value)
	}
	return api.ExpressionEmission{}, false, nil
}

func direct(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	target, err := api.NewExpressionEmission(
		value.Before(),
		context.Factory().BinaryExpression(
			nil,
			value.Value(),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			context.Factory().Identifier("undefined"),
		),
		value.Requests(),
	)
	return target, true, err
}

func method(
	context api.Context,
	value api.ExpressionEmission,
	name string,
	err error,
) (api.ExpressionEmission, bool, error) {
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if name == "" {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "nil-comparison runtime member is invalid",
		}
	}
	target, buildErr := api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				value.Value(),
				nil,
				context.Factory().Identifier(name),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		),
		value.Requests(),
	)
	return target, true, buildErr
}
