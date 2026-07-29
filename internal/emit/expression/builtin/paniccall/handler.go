package paniccall

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	panicnilruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panicnil"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, bool, error) {
	if types.Object(builtin) != types.Universe.Lookup("panic") {
		return api.ExpressionEmission{}, false, nil
	}
	if source == nil || len(source.Args) != 1 || source.Ellipsis.IsValid() {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emptyInterface := types.NewInterfaceType(nil, nil).Complete()
	argument, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(emptyInterface),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	argument, err = captureArgument(context, argument)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	nilReference, err := context.Names().Runtime(
		api.RuntimePanicNilValue,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	payload := context.Factory().ConditionalExpression(
		context.Factory().BinaryExpression(
			nil,
			argument.Value(),
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			context.Factory().Identifier("undefined"),
		),
		context.Factory().QuestionToken(),
		panicnilruntime.Create(
			context.Factory(),
			nilReference.Name(),
		),
		context.Factory().ColonToken(),
		argument.Value(),
	)
	target, err := api.NewExpressionEmission(
		argument.Before(),
		panicruntime.CallValue(
			context.Factory(),
			panicReference.Name(),
			payload,
		),
		api.CombineRequests(
			argument.Requests(),
			panicReference.Requests(),
			nilReference.Requests(),
		),
	)
	return target, true, err
}

func EmitDeferred(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, bool, error) {
	if types.Object(builtin) != types.Universe.Lookup("panic") {
		return api.ExpressionEmission{}, false, nil
	}
	if source == nil || len(source.Args) != 1 || source.Ellipsis.IsValid() {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	emptyInterface := types.NewInterfaceType(nil, nil).Complete()
	argument, err := children.Expression(
		context.
			WithRole(api.RoleBuiltinArgument).
			WithExpectedType(emptyInterface),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	argument, err = captureArgument(context, argument)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	panicReference, err := context.Names().Runtime(
		api.RuntimePanic,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	nilReference, err := context.Names().Runtime(
		api.RuntimePanicNilValue,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	value := argument.Value()
	payload := context.Factory().ConditionalExpression(
		context.Factory().BinaryExpression(
			nil,
			value,
			nil,
			context.Factory().BinaryOperatorToken(
				tsgo.BinaryOperatorEqualsEqualsEqualsToken,
			),
			context.Factory().Identifier("undefined"),
		),
		context.Factory().QuestionToken(),
		panicnilruntime.Create(
			context.Factory(),
			nilReference.Name(),
		),
		context.Factory().ColonToken(),
		value,
	)
	target, err := api.NewExpressionEmission(
		argument.Before(),
		panicruntime.CallValue(
			context.Factory(),
			panicReference.Name(),
			payload,
		),
		api.CombineRequests(
			argument.Requests(),
			panicReference.Requests(),
			nilReference.Requests(),
		),
	)
	return target, true, err
}

func captureArgument(
	context api.Context,
	argument api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	name, err := context.Names().Temporary(api.TemporaryCallArgument)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before := append(
		argument.Before(),
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						argument.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	return api.NewExpressionEmission(
		before,
		context.Factory().Identifier(name),
		argument.Requests(),
	)
}
