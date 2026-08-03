package builtin

import (
	"go/ast"
	"go/constant"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	runtimeslice "github.com/tsoniclang/gotots/internal/emit/runtime/slice"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
	arrayvalue "github.com/tsoniclang/gotots/internal/emit/value/array"
	"github.com/tsoniclang/gotots/internal/emit/value/maprepresentation"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitGenericMeasure(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	if source == nil ||
		builtin == nil ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{}, false, nil
	}
	operation := api.GenericOperationInvalid
	switch types.Object(builtin) {
	case types.Universe.Lookup("len"):
		operation = api.GenericOperationLength
	case types.Universe.Lookup("cap"):
		operation = api.GenericOperationCapacity
	default:
		return api.ExpressionEmission{}, false, nil
	}
	operandType := context.TypesInfo().TypeOf(source.Args[0])
	if !api.ContainsGenericTypeParameter(operandType) {
		return api.ExpressionEmission{}, false, nil
	}
	resultType := context.TypesInfo().TypeOf(source)
	if resultType == nil ||
		(!discarded &&
			(context.ExpectedType() == nil ||
				!types.AssignableTo(resultType, context.ExpectedType()))) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(operandType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := genericoperation.Call(
		context,
		source,
		operation,
		[]types.Type{operandType},
		[]types.Type{resultType},
		[]api.ExpressionEmission{operand},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return target, true, nil
}

func ApplyMeasure(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	operation api.GenericOperation,
	operandType types.Type,
	resultType types.Type,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if !basictype.SupportsInteger(context.TypesSizes(), resultType) {
		return api.ExpressionEmission{}, false, nil
	}
	member := runtimeslice.MemberInvalid
	switch operation {
	case api.GenericOperationLength:
		member = runtimeslice.MemberLength
	case api.GenericOperationCapacity:
		member = runtimeslice.MemberCapacity
	default:
		return api.ExpressionEmission{}, false, nil
	}
	if supportsStringArgument(operandType) {
		if operation != api.GenericOperationLength {
			return api.ExpressionEmission{}, false, nil
		}
		defined, definedArgument := definedtype.ResolveBasic(operandType)
		var err error
		if definedArgument {
			operand, err = defined.Project(context, operand)
			if err != nil {
				return api.ExpressionEmission{}, true, err
			}
		}
		target, err := measuredProperty(context, operand, "length")
		return target, true, err
	}
	if array, ok := arrayvalue.Resolve(context, operandType); ok {
		target, err := array.Measure(context, operand)
		return target, true, err
	}
	if pointed, ok := pointedArray(context, operandType); ok {
		return constantArrayMeasure(
			context,
			source,
			resultType,
			pointed.Length(),
			operand,
		)
	}
	if _, _, ok := slicevalue.Source(operandType); ok {
		var err error
		operand, err = slicevalue.Project(context, operandType, operand)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		target, err := measuredProperty(
			context,
			operand,
			runtimeslice.MemberName(member),
		)
		return target, true, err
	}
	if mapType, ok := maprepresentation.Source(context, operandType); ok {
		if operation != api.GenericOperationLength {
			return api.ExpressionEmission{}, false, nil
		}
		var err error
		operand, err = mapType.ReadReceiver(context, source, operand)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		length, err := mapruntime.Name(mapruntime.MemberLength)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		target, err := api.NewExpressionEmission(
			operand.Before(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					operand.Value(),
					nil,
					context.Factory().Identifier(length),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
			operand.Requests(),
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		target, err = representMeasure(context, target)
		return target, true, err
	}
	return api.ExpressionEmission{}, false, nil
}

func measuredProperty(
	context api.Context,
	operand api.ExpressionEmission,
	member string,
) (api.ExpressionEmission, error) {
	target, err := api.NewExpressionEmission(
		operand.Before(),
		context.Factory().PropertyAccessExpression(
			operand.Value(),
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		operand.Requests(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return representMeasure(context, target)
}

func representMeasure(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if context.IntegerRepresentation() != api.IntegerRepresentationBigInt {
		return value, nil
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().CallExpression(
			api.TargetIntrinsicBigInt.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{value.Value()},
			tsgo.NodeFlagsNone,
		),
		value.Requests(),
	)
}

func pointedArray(
	context api.Context,
	sourceType types.Type,
) (arrayvalue.RuntimeArray, bool) {
	_, element, ok := pointertype.Resolve(sourceType)
	if !ok {
		if defined, definedOK := definedtype.ResolvePointer(sourceType); definedOK {
			pointer, _ := defined.Pointer()
			element = pointer.Elem()
			ok = true
		}
	}
	if !ok {
		return arrayvalue.RuntimeArray{}, false
	}
	return arrayvalue.Resolve(context, element)
}

func constantArrayMeasure(
	context api.Context,
	source ast.Node,
	resultType types.Type,
	length int64,
	operand api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	name, err := context.Names().Temporary(api.TemporaryCallArgument)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := constantvalue.EmitValue(
		context.WithRole(api.RoleBuiltinArgument),
		source,
		resultType,
		constant.MakeInt64(length),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	before := append(
		operand.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						operand.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	before = append(before, target.Before()...)
	target, err = api.NewExpressionEmission(
		before,
		target.Value(),
		api.CombineRequests(operand.Requests(), target.Requests()),
	)
	return target, true, err
}
