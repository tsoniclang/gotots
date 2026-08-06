package defined

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	basicbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/basic"
	complexbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/complex"
	floatbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/float"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
	expressionoperands "github.com/tsoniclang/gotots/internal/emit/expression/operands"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	model, wrapsResult, ok := operationModel(context, source)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	operationContext, err := model.OperationContext(context)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	left, err := operand(
		context.WithRole(api.RoleBinaryLeft),
		operationContext.WithRole(api.RoleBinaryLeft),
		children,
		source.X,
		model,
		comparison(source.Op),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := operand(
		context.WithRole(api.RoleBinaryRight),
		operationContext.WithRole(api.RoleBinaryRight),
		children,
		source.Y,
		model,
		comparison(source.Op),
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
	underlying, valid := model.Basic()
	if !valid {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined basic operation has no basic underlying type",
		}
	}
	target, handled, err := apply(
		operationContext,
		source.Op,
		underlying,
		context.TypesInfo().TypeOf(source.Y),
		operands.Left(),
		operands.Right(),
		rightConstant(context, source.Y),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !handled {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if wrapsResult {
		target, err = model.Wrap(context, target)
	}
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, err = expressionoperands.Finish(operands, target)
	return target, true, err
}

func operationModel(
	context api.Context,
	source *ast.BinaryExpr,
) (definedtype.Model, bool, bool) {
	if source == nil {
		return definedtype.Model{}, false, false
	}
	if model, ok := definedtype.ResolveBasic(context.TypesInfo().TypeOf(source)); ok {
		return model, true, operandsBelong(context, source, model)
	}
	if source.Op == token.SHL || source.Op == token.SHR {
		resultType := context.TypesInfo().TypeOf(source)
		expected := context.ExpectedType()
		model, ok := definedtype.ResolveBasic(expected)
		if ok &&
			constantvalue.IsUntyped(resultType) &&
			types.AssignableTo(resultType, expected) &&
			operandsBelong(context, source, model) {
			return model, true, true
		}
	}
	if !comparison(source.Op) {
		return definedtype.Model{}, false, false
	}
	for _, candidate := range []types.Type{
		context.TypesInfo().TypeOf(source.X),
		context.TypesInfo().TypeOf(source.Y),
	} {
		model, ok := definedtype.ResolveBasic(candidate)
		if ok && operandsBelong(context, source, model) {
			return model, false, true
		}
	}
	return definedtype.Model{}, false, false
}

func operandsBelong(
	context api.Context,
	source *ast.BinaryExpr,
	model definedtype.Model,
) bool {
	left := context.TypesInfo().TypeOf(source.X)
	right := context.TypesInfo().TypeOf(source.Y)
	if source.Op == token.SHL || source.Op == token.SHR {
		return types.AssignableTo(left, model.Type())
	}
	return types.AssignableTo(left, model.Type()) &&
		types.AssignableTo(right, model.Type())
}

func operand(
	context api.Context,
	operationContext api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	model definedtype.Model,
	wrapLiteral bool,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if literalConstant(source) {
		facts, ok := context.TypesInfo().TypeAndValue(source)
		if ok && facts.Value != nil {
			target, err := constantvalue.EmitValue(
				operationContext,
				source,
				model.Underlying(),
				facts.Value,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			representation, err := model.Representation(context)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
			if wrapLiteral && representation.Kind() ==
				api.DefinedValueRepresentationGeneratedNumeric {
				return model.Wrap(context, target)
			}
			return target, nil
		}
	}
	expected := sourceType
	projectContextual := false
	if types.AssignableTo(sourceType, model.Type()) {
		expected = model.Type()
		projectContextual = true
	}
	target, err := children.Expression(
		context.WithExpectedType(expected),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if sourceModel, ok := definedtype.ResolveBasic(sourceType); ok {
		target, err = sourceModel.Project(context, target)
	} else if projectContextual {
		target, err = model.Project(context, target)
	}
	return target, err
}

func literalConstant(source ast.Expr) bool {
	switch source := source.(type) {
	case *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return literalConstant(source.X)
	case *ast.UnaryExpr:
		switch source.Op {
		case token.ADD, token.SUB, token.XOR, token.NOT:
			return literalConstant(source.X)
		default:
			return false
		}
	default:
		return false
	}
}

func apply(
	context api.Context,
	operator token.Token,
	underlying *types.Basic,
	rightType types.Type,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
	rightConstant constant.Value,
) (api.ExpressionEmission, bool, error) {
	if carrier, ok := integervalue.Describe(
		context.TypesSizes(),
		underlying,
	); ok {
		rightCarrier := carrier
		if operator == token.SHL || operator == token.SHR {
			if basic, ok := types.Unalias(rightType).(*types.Basic); ok &&
				basic.Info()&types.IsUntyped != 0 {
				rightType = types.Default(rightType)
			}
			var rightOK bool
			rightCarrier, rightOK = integervalue.DescribeUnderlying(
				context.TypesSizes(),
				rightType,
			)
			if !rightOK {
				return api.ExpressionEmission{}, false, nil
			}
		}
		if (operator == token.SHL || operator == token.SHR) &&
			rightConstant == nil {
			if !integervalue.SupportsVariableShift(
				context.IntegerRepresentation(),
				carrier,
				operator,
			) {
				return api.ExpressionEmission{}, false, nil
			}
			return integerbinary.ApplyVariableShift(
				context,
				operator,
				carrier,
				rightCarrier,
				left,
				right,
			)
		}
		if !integerOperation(
			context,
			operator,
			carrier,
			rightConstant,
		) {
			return api.ExpressionEmission{}, false, nil
		}
		return integerbinary.Apply(
			context,
			operator,
			carrier,
			rightCarrier,
			left,
			right,
		)
	}
	if carrier, ok := floatvalue.Describe(underlying); ok {
		return floatbinary.Apply(
			context,
			operator,
			carrier,
			left,
			right,
		)
	}
	if carrier, ok := complexvalue.Describe(underlying); ok {
		return complexbinary.Apply(
			context,
			operator,
			carrier,
			left,
			right,
		)
	}
	target, handled := basicbinary.Apply(
		context,
		underlying,
		operator,
		left,
		right,
	)
	return target, handled, nil
}

func ApplyUnderlying(
	context api.Context,
	operator token.Token,
	underlying *types.Basic,
	rightType types.Type,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
	rightConstant constant.Value,
) (api.ExpressionEmission, bool, error) {
	return apply(
		context,
		operator,
		underlying,
		rightType,
		left,
		right,
		rightConstant,
	)
}

func integerOperation(
	context api.Context,
	operator token.Token,
	carrier integervalue.Carrier,
	rightConstant constant.Value,
) bool {
	switch {
	case comparison(operator):
		return true
	case integervalue.SupportsArithmetic(
		context.IntegerRepresentation(),
		operator,
	):
		return true
	case integervalue.SupportsBitwise(
		context.IntegerRepresentation(),
		carrier,
		operator,
	):
		return true
	case operator == token.SHL || operator == token.SHR:
		return integervalue.SupportsShift(
			context.IntegerRepresentation(),
			carrier,
			operator,
			rightConstant,
		)
	default:
		return false
	}
}

func rightConstant(context api.Context, source ast.Expr) constant.Value {
	facts, ok := context.TypesInfo().TypeAndValue(source)
	if !ok {
		return nil
	}
	return facts.Value
}

func comparison(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}
