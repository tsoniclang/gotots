package slice

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	slicingexpression "github.com/tsoniclang/gotots/internal/emit/expression/slicing"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	integeroperand "github.com/tsoniclang/gotots/internal/emit/value/integer/operand"
	slicevalue "github.com/tsoniclang/gotots/internal/emit/value/slice"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
) (api.ExpressionEmission, error) {
	operandType := context.TypesInfo().TypeOf(source.X)
	if _, generic := api.GenericTypeParameter(operandType); generic {
		return emitGeneric(context, children, source, operandType)
	}
	if target, handled, err := slicingexpression.EmitArray(
		context,
		children,
		source,
	); handled || err != nil {
		return target, err
	}
	if _, _, ok := slicevalue.Resolve(operandType); ok {
		return slicingexpression.Emit(context, children, source)
	}
	if _, ok := definedtype.ResolveSlice(operandType); ok {
		return slicingexpression.Emit(context, children, source)
	}
	resultType := context.TypesInfo().TypeOf(source)
	stringType := operandType
	operandExpectedType := types.Type(types.Typ[types.String])
	definedString, definedStringOK := definedtype.ResolveBasic(operandType)
	if definedStringOK {
		underlying, basicOK := definedString.Basic()
		resultDefined, resultDefinedOK := definedtype.ResolveBasic(resultType)
		if !basicOK ||
			!basictype.SupportsString(underlying) ||
			!resultDefinedOK ||
			resultDefined.TypeName() != definedString.TypeName() {
			definedStringOK = false
		} else {
			stringType = underlying
			operandExpectedType = operandType
		}
	}
	if source.Slice3 ||
		source.Max != nil ||
		!basictype.SupportsString(stringType) ||
		(!definedStringOK && !basictype.SupportsString(resultType)) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if expected := context.ExpectedType(); expected != nil &&
		!types.AssignableTo(resultType, expected) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleSliceOperand).
			WithExpectedType(operandExpectedType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedStringOK {
		operand, err = definedString.Project(context, operand)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	low, err := bound(
		context.WithRole(api.RoleSliceLow),
		children,
		source,
		source.Low,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	operands := []expressionoperands.Item{
		expressionoperands.Present(operand),
		expressionoperands.Present(low),
	}
	var high api.ExpressionEmission
	if source.High != nil {
		high, err = bound(
			context.WithRole(api.RoleSliceHigh),
			children,
			source,
			source.High,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		operands = append(operands, expressionoperands.Present(high))
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		operands...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	reference, err := context.Names().Runtime(
		api.RuntimeStringSlice,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	target, err := api.NewExpressionEmission(
		ordered.Before(),
		context.Factory().CallExpression(
			reference.Expression(context.Factory()),
			nil,
			nil,
			ordered.Values(),
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			ordered.Requests(),
			reference.Requests(),
		),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if definedStringOK {
		return definedString.Wrap(context, target)
	}
	return target, nil
}

type genericBound struct {
	emission   api.ExpressionEmission
	sourceType types.Type
}

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
	operandType types.Type,
) (api.ExpressionEmission, error) {
	resultType := context.TypesInfo().TypeOf(source)
	if resultType == nil ||
		context.ExpectedType() == nil ||
		!types.AssignableTo(resultType, context.ExpectedType()) ||
		(source.Slice3 && source.Max == nil) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	receiver, err := children.Expression(
		context.
			WithRole(api.RoleSliceOperand).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverName, err := context.Names().Temporary(api.TemporarySliceOperand)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	receiverValue := context.Factory().Identifier(receiverName)
	before := append([]tsgo.Statement(nil), receiver.Before()...)
	before = append(before, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				receiverValue,
				nil,
				nil,
				receiver.Value(),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	low, err := emitGenericBound(
		context.WithRole(api.RoleSliceLow),
		children,
		source.Low,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var high genericBound
	if source.High == nil {
		highValue, highErr := genericoperation.Call(
			context.WithRole(api.RoleSliceHigh),
			source,
			api.GenericOperationLength,
			[]types.Type{operandType},
			[]types.Type{types.Typ[types.Int]},
			[]api.ExpressionEmission{api.DirectExpression(receiverValue)},
		)
		if highErr != nil {
			return api.ExpressionEmission{}, highErr
		}
		high = genericBound{
			emission:   highValue,
			sourceType: types.Typ[types.Int],
		}
	} else {
		high, err = emitGenericBound(
			context.WithRole(api.RoleSliceHigh),
			children,
			source.High,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	bounds := []genericBound{low, high}
	operation := api.GenericOperationSlice
	if source.Slice3 {
		maximum, maxErr := emitGenericBound(
			context.WithRole(api.RoleSliceMax),
			children,
			source.Max,
		)
		if maxErr != nil {
			return api.ExpressionEmission{}, maxErr
		}
		bounds = append(bounds, maximum)
		operation = api.GenericOperationSliceFull
	}
	items := make([]expressionoperands.Item, 0, len(bounds))
	parameterTypes := []types.Type{operandType}
	for _, bound := range bounds {
		items = append(items, expressionoperands.Present(bound.emission))
		parameterTypes = append(parameterTypes, bound.sourceType)
	}
	ordered, err := expressionoperands.Preserve(
		context,
		api.TemporarySliceOperand,
		items...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	arguments := []api.ExpressionEmission{api.DirectExpression(receiverValue)}
	for _, value := range ordered.Values() {
		arguments = append(arguments, api.DirectExpression(value))
	}
	target, err := genericoperation.Call(
		context,
		source,
		operation,
		parameterTypes,
		[]types.Type{resultType},
		arguments,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, ordered.Before()...)
	before = append(before, target.Before()...)
	return api.NewExpressionEmission(
		before,
		target.Value(),
		api.CombineRequests(
			receiver.Requests(),
			ordered.Requests(),
			target.Requests(),
		),
	)
}

func emitGenericBound(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (genericBound, error) {
	if source == nil {
		zero, err := api.IntegerLiteral(
			context.Factory(),
			context.IntegerRepresentation(),
			"0",
		)
		return genericBound{
			emission:   api.DirectExpression(zero),
			sourceType: types.Typ[types.Int],
		}, err
	}
	value, err := integeroperand.Emit(context, children, source)
	if err != nil {
		return genericBound{}, err
	}
	return genericBound{
		emission:   value,
		sourceType: representedBoundType(context.TypesInfo().TypeOf(source)),
	}, nil
}

func representedBoundType(sourceType types.Type) types.Type {
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 {
		return types.Typ[types.Int]
	}
	if defined, ok := definedtype.ResolveBasic(sourceType); ok {
		return defined.Underlying()
	}
	return sourceType
}

func bound(
	context api.Context,
	children api.ChildEmitter,
	source *ast.SliceExpr,
	value ast.Expr,
) (api.ExpressionEmission, error) {
	if value == nil {
		zero, err := api.IntegerLiteral(
			context.Factory(),
			context.IntegerRepresentation(),
			"0",
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.DirectExpression(zero), nil
	}
	return integeroperand.Emit(context, children, value)
}
