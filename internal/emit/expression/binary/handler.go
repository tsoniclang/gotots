package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	basicbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/basic"
	complexbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/complex"
	definedbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/defined"
	floatbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/float"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/emit/value/nilcomparison"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, error) {
	if target, handled, err := constantvalue.EmitFolded(
		context,
		source,
	); handled {
		return target, err
	}
	if binaryConstantEvidenceIsIncomplete(context, source) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if target, handled, err := nilcomparison.Emit(
		context,
		children,
		source,
	); handled {
		return target, err
	}
	if target, handled, err := emitGeneric(
		context,
		children,
		source,
	); handled {
		return target, err
	}
	if source.Op == token.EQL || source.Op == token.NEQ {
		if target, ok, err := emitValueEquality(context, children, source); ok || err != nil {
			return target, err
		}
	}
	if target, ok, err := definedbinary.Emit(
		context,
		children,
		source,
	); ok || err != nil {
		return target, err
	}
	if target, ok, err := complexbinary.Emit(
		context,
		children,
		source,
	); ok || err != nil {
		return target, err
	}
	if target, ok, err := integerbinary.Emit(
		context,
		children,
		source,
	); ok || err != nil {
		return target, err
	}
	if target, ok, err := floatbinary.Emit(
		context,
		children,
		source,
	); ok || err != nil {
		return target, err
	}
	operator, operandType, ok := operationFor(context, source)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	left, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	right, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(operandType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if isLogicalOperator(source.Op) {
		return emitLogical(context, source.Op, operator, left, right)
	}
	operands, err := expressionoperands.PreservePair(
		context,
		left,
		right,
		api.TemporaryBinaryOperand,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return expressionoperands.Finish(
		operands,
		api.DirectExpression(
			context.Factory().BinaryExpression(
				nil,
				operands.Left().Value(),
				nil,
				operator,
				operands.Right().Value(),
			),
			api.CombineRequests(
				operands.Left().Requests(),
				operands.Right().Requests(),
			)...,
		),
	)
}

func emitGeneric(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	if target, handled, err := emitGenericNilEquality(
		context,
		children,
		source,
	); handled {
		return target, true, err
	}
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	resultType := context.TypesInfo().TypeOf(source)
	_, leftGeneric := api.GenericTypeParameter(leftType)
	_, rightGeneric := api.GenericTypeParameter(rightType)
	if !leftGeneric && !rightGeneric {
		return api.ExpressionEmission{}, false, nil
	}
	operation, ok := api.BinaryGenericOperation(source.Op)
	if !ok || resultType == nil || isLogicalOperator(source.Op) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	switch source.Op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		resultType = types.Typ[types.Bool]
	}
	leftType, rightType = contextualGenericOperandTypes(
		leftType,
		rightType,
		leftGeneric,
		rightGeneric,
	)
	left, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(leftType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(rightType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	operands, err := expressionoperands.PreservePair(
		context,
		left,
		right,
		api.TemporaryBinaryOperand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := genericoperation.Call(
		context,
		source,
		operation,
		[]types.Type{leftType, rightType},
		[]types.Type{resultType},
		[]api.ExpressionEmission{
			operands.Left(),
			operands.Right(),
		},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result, err := expressionoperands.Finish(operands, target)
	return result, true, err
}

func emitGenericNilEquality(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	valueSource, role, valueType, negated, selected :=
		nilcomparison.SelectSource(context.TypesInfo(), source)
	if !selected {
		return api.ExpressionEmission{}, false, nil
	}
	parameter, generic := api.GenericTypeParameter(valueType)
	if !generic {
		return api.ExpressionEmission{}, false, nil
	}
	value, err := children.Expression(
		context.WithRole(role).WithExpectedType(parameter),
		valueSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err := genericoperation.Call(
		context,
		source,
		api.GenericOperationNilEqual,
		[]types.Type{parameter},
		[]types.Type{types.Typ[types.Bool]},
		[]api.ExpressionEmission{value},
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result := target.Value()
	if negated {
		result = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			result,
		)
	}
	emission, err := api.NewExpressionEmission(
		target.Before(),
		result,
		target.Requests(),
	)
	return emission, true, err
}

func contextualGenericOperandTypes(
	leftType types.Type,
	rightType types.Type,
	leftGeneric bool,
	rightGeneric bool,
) (types.Type, types.Type) {
	switch {
	case leftGeneric &&
		constantvalue.IsUntyped(rightType) &&
		types.AssignableTo(rightType, leftType):
		rightType = leftType
	case rightGeneric &&
		constantvalue.IsUntyped(leftType) &&
		types.AssignableTo(leftType, rightType):
		leftType = rightType
	}
	return leftType, rightType
}

func emitLogical(
	context api.Context,
	sourceOperator token.Token,
	targetOperator tsgo.BinaryOperatorToken,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if len(right.Before()) == 0 {
		return api.NewExpressionEmission(
			left.Before(),
			context.Factory().BinaryExpression(
				nil,
				left.Value(),
				nil,
				targetOperator,
				right.Value(),
			),
			api.CombineRequests(left.Requests(), right.Requests()),
		)
	}
	resultName, err := context.Names().Temporary(api.TemporaryLogicalResult)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result := context.Factory().Identifier(resultName)
	condition := tsgo.Expression(result)
	if sourceOperator == token.LOR {
		condition = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			result,
		)
	}
	branch := right.Before()
	branch = append(
		branch,
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				result,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				right.Value(),
			),
		),
	)
	before := left.Before()
	before = append(
		before,
		binaryVariable(
			context,
			tsgo.NodeFlagsLet,
			resultName,
			left.Value(),
		),
		context.Factory().IfStatement(
			condition,
			context.Factory().Block(branch, true),
			nil,
		),
	)
	return api.NewExpressionEmission(
		before,
		result,
		api.CombineRequests(left.Requests(), right.Requests()),
	)
}

func binaryConstantEvidenceIsIncomplete(
	context api.Context,
	source *ast.BinaryExpr,
) bool {
	result, resultExists := context.TypesInfo().Types[source]
	left, leftExists := context.TypesInfo().Types[source.X]
	right, rightExists := context.TypesInfo().Types[source.Y]
	return resultExists &&
		result.Value == nil &&
		leftExists &&
		left.Value != nil &&
		rightExists &&
		right.Value != nil
}

func emitValueEquality(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	if leftType == nil || rightType == nil {
		return api.ExpressionEmission{}, false, nil
	}
	var operandType types.Type
	for _, candidate := range []types.Type{leftType, rightType} {
		if context.Values().RequiresCustomEquality(context, candidate) &&
			types.AssignableTo(leftType, candidate) &&
			types.AssignableTo(rightType, candidate) {
			operandType = candidate
			break
		}
	}
	if operandType == nil {
		return api.ExpressionEmission{}, false, nil
	}
	left, err := children.Expression(
		context.
			WithRole(api.RoleBinaryLeft).
			WithExpectedType(operandType),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	left, err = context.Values().Transfer(
		context.WithRole(api.RoleBinaryLeft),
		source.X,
		leftType,
		operandType,
		api.ValueTransferRepresentation,
		left,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := children.Expression(
		context.
			WithRole(api.RoleBinaryRight).
			WithExpectedType(operandType),
		source.Y,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err = context.Values().Transfer(
		context.WithRole(api.RoleBinaryRight),
		source.Y,
		rightType,
		operandType,
		api.ValueTransferRepresentation,
		right,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	before := left.Before()
	leftValue := left.Value()
	if len(right.Before()) != 0 {
		leftName, err := context.Names().Temporary(
			api.TemporaryEqualityOperand,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		before = append(
			before,
			equalityOperandStatement(context, leftName, leftValue),
		)
		leftValue = context.Factory().Identifier(leftName)
	}
	before = append(before, right.Before()...)
	equal, err := context.Values().Equal(
		context,
		source,
		operandType,
		leftValue,
		right.Value(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	before = append(before, equal.Before()...)
	value := equal.Value()
	if source.Op == token.NEQ {
		value = context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			value,
		)
	}
	result, err := api.NewExpressionEmission(
		before,
		value,
		api.CombineRequests(
			left.Requests(),
			right.Requests(),
			equal.Requests(),
		),
	)
	return result, true, err
}

func equalityOperandStatement(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return binaryVariable(context, tsgo.NodeFlagsConst, name, value)
}

func binaryVariable(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					nil,
					value,
				),
			},
			flags,
		),
	)
}

func operationFor(
	context api.Context,
	source *ast.BinaryExpr,
) (tsgo.BinaryOperatorToken, types.Type, bool) {
	leftType := context.TypesInfo().TypeOf(source.X)
	rightType := context.TypesInfo().TypeOf(source.Y)
	switch {
	case source.Op == token.ADD &&
		basictype.SupportsString(context.TypesInfo().TypeOf(source)) &&
		types.AssignableTo(leftType, context.TypesInfo().TypeOf(source)) &&
		types.AssignableTo(rightType, context.TypesInfo().TypeOf(source)):
		operator, ok := basicbinary.Operator(
			context,
			context.TypesInfo().TypeOf(source),
			source.Op,
		)
		return operator, context.TypesInfo().TypeOf(source), ok
	case isStringComparison(source.Op) &&
		basictype.SupportsString(leftType) &&
		basictype.SupportsString(rightType):
		operator, ok := basicbinary.Operator(
			context,
			types.Typ[types.String],
			source.Op,
		)
		return operator, types.Typ[types.String], ok
	case isLogicalOperator(source.Op) &&
		isSupportedBoolean(context.TypesInfo().TypeOf(source)) &&
		types.AssignableTo(leftType, types.Typ[types.Bool]) &&
		types.AssignableTo(rightType, types.Typ[types.Bool]):
		operator, ok := basicbinary.Operator(
			context,
			types.Typ[types.Bool],
			source.Op,
		)
		return operator, types.Typ[types.Bool], ok
	case source.Op == token.EQL &&
		types.AssignableTo(leftType, types.Typ[types.Bool]) &&
		types.AssignableTo(rightType, types.Typ[types.Bool]):
		operator, ok := basicbinary.Operator(
			context,
			types.Typ[types.Bool],
			source.Op,
		)
		return operator, types.Typ[types.Bool], ok
	case source.Op == token.NEQ &&
		types.AssignableTo(leftType, types.Typ[types.Bool]) &&
		types.AssignableTo(rightType, types.Typ[types.Bool]):
		operator, ok := basicbinary.Operator(
			context,
			types.Typ[types.Bool],
			source.Op,
		)
		return operator, types.Typ[types.Bool], ok
	default:
		return nil, nil, false
	}
}

func isStringComparison(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func isLogicalOperator(operator token.Token) bool {
	return operator == token.LAND || operator == token.LOR
}

func isSupportedBoolean(value types.Type) bool {
	basic, ok := types.Unalias(value).(*types.Basic)
	return ok && basic.Info()&types.IsBoolean != 0
}
