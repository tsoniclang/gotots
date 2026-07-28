package defined

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	basicbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/basic"
	complexbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/complex"
	floatbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/float"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
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
	left, err := operand(
		context.WithRole(api.RoleBinaryLeft),
		children,
		source.X,
		model,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	right, err := operand(
		context.WithRole(api.RoleBinaryRight),
		children,
		source.Y,
		model,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if len(left.Before()) != 0 || len(right.Before()) != 0 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, handled, err := apply(
		context,
		source,
		model.Underlying(),
		left,
		right,
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
	return target, true, err
}

func operationModel(
	context api.Context,
	source *ast.BinaryExpr,
) (definedtype.Model, bool, bool) {
	if source == nil {
		return definedtype.Model{}, false, false
	}
	if model, ok := definedtype.Resolve(context.TypesInfo().TypeOf(source)); ok {
		return model, true, operandsBelong(context, source, model)
	}
	if !comparison(source.Op) {
		return definedtype.Model{}, false, false
	}
	for _, candidate := range []types.Type{
		context.TypesInfo().TypeOf(source.X),
		context.TypesInfo().TypeOf(source.Y),
	} {
		model, ok := definedtype.Resolve(candidate)
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
	children api.ChildEmitter,
	source ast.Expr,
	model definedtype.Model,
) (api.ExpressionEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if literalConstant(source) {
		facts, ok := context.TypesInfo().Types[source]
		if ok && facts.Value != nil {
			return constantvalue.EmitValue(
				context,
				source,
				model.Underlying(),
				facts.Value,
			)
		}
	}
	expected := sourceType
	if types.AssignableTo(sourceType, model.Type()) {
		expected = model.Type()
	}
	target, err := children.Expression(
		context.WithExpectedType(expected),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if sourceModel, ok := definedtype.Resolve(sourceType); ok {
		target, err = api.NewExpressionEmission(
			target.Before(),
			sourceModel.Unwrap(context.Factory(), target.Value()),
			target.Requests(),
		)
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
	source *ast.BinaryExpr,
	underlying *types.Basic,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	if carrier, ok := integervalue.Describe(
		context.TypesSizes(),
		underlying,
	); ok {
		if !integerOperation(context, source, carrier) {
			return api.ExpressionEmission{}, false, nil
		}
		return integerbinary.Apply(
			context,
			source.Op,
			carrier,
			left,
			right,
		)
	}
	if carrier, ok := floatvalue.Describe(underlying); ok {
		return floatbinary.Apply(
			context,
			source.Op,
			carrier,
			left,
			right,
		)
	}
	if carrier, ok := complexvalue.Describe(underlying); ok {
		return complexbinary.Apply(
			context,
			source.Op,
			carrier,
			left,
			right,
		)
	}
	target, handled := basicbinary.Apply(
		context,
		underlying,
		source.Op,
		left,
		right,
	)
	return target, handled, nil
}

func integerOperation(
	context api.Context,
	source *ast.BinaryExpr,
	carrier integervalue.Carrier,
) bool {
	switch {
	case comparison(source.Op):
		return true
	case integervalue.SupportsArithmetic(
		context.IntegerRepresentation(),
		source.Op,
	):
		return true
	case integervalue.SupportsBitwise(
		context.IntegerRepresentation(),
		carrier,
		source.Op,
	):
		return true
	case source.Op == token.SHL || source.Op == token.SHR:
		facts, ok := context.TypesInfo().Types[source.Y]
		return ok && integervalue.SupportsShift(
			context.IntegerRepresentation(),
			carrier,
			source.Op,
			facts.Value,
		)
	default:
		return false
	}
}

func comparison(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}
