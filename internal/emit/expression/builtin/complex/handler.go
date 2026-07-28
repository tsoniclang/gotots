package complex

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	runtimecomplex "github.com/tsoniclang/gotots/internal/emit/runtime/complex"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
	discarded bool,
) (api.ExpressionEmission, bool, error) {
	switch types.Object(builtin) {
	case types.Universe.Lookup("complex"):
		target, err := emitConstruct(context, children, source, discarded)
		return target, true, err
	case types.Universe.Lookup("real"):
		target, err := emitComponent(
			context,
			children,
			source,
			discarded,
			runtimecomplex.RealMember,
		)
		return target, true, err
	case types.Universe.Lookup("imag"):
		target, err := emitComponent(
			context,
			children,
			source,
			discarded,
			runtimecomplex.ImagMember,
		)
		return target, true, err
	default:
		return api.ExpressionEmission{}, false, nil
	}
}

func emitConstruct(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	resultType := context.TypesInfo().TypeOf(source)
	expectedType := context.ExpectedType()
	sourceFacts, factsOK := context.TypesInfo().Types[source]
	if discarded ||
		len(source.Args) != 2 ||
		expectedType == nil ||
		resultType == nil ||
		!types.AssignableTo(resultType, expectedType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if factsOK && sourceFacts.Value != nil {
		if _, ok := complexvalue.Describe(expectedType); !ok {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
		return constantvalue.EmitValue(
			context,
			source,
			expectedType,
			sourceFacts.Value,
		)
	}
	carrier, ok := complexvalue.Describe(resultType)
	if discarded ||
		!ok ||
		!types.AssignableTo(resultType, expectedType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	componentType := carrier.ComponentType()
	first, err := children.Expression(
		context.WithRole(api.RoleCallArgument).WithExpectedType(componentType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	second, err := children.Expression(
		context.WithRole(api.RoleCallArgument).WithExpectedType(componentType),
		source.Args[1],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	firstValue, before, err := orderedFirst(
		context,
		first,
		second,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, second.Before()...)
	target, err := complexvalue.Construct(
		context,
		carrier,
		firstValue,
		second.Value(),
		api.CombineRequests(first.Requests(), second.Requests())...,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		before,
		target.Value(),
		target.Requests(),
	)
}

func orderedFirst(
	context api.Context,
	first api.ExpressionEmission,
	second api.ExpressionEmission,
) (tsgo.Expression, []tsgo.Statement, error) {
	before := first.Before()
	if context.EvaluationOrder() != api.EvaluationOrderPreserveGo ||
		len(second.Before()) == 0 {
		return first.Value(), before, nil
	}
	name, err := context.Names().Temporary(api.TemporaryCallArgument)
	if err != nil {
		return nil, nil, err
	}
	before = append(
		before,
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						context.Factory().Identifier(name),
						nil,
						nil,
						first.Value(),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	)
	return context.Factory().Identifier(name), before, nil
}

func emitComponent(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	member string,
) (api.ExpressionEmission, error) {
	if discarded || len(source.Args) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	resultType := context.TypesInfo().TypeOf(source)
	expectedType := context.ExpectedType()
	sourceFacts, factsOK := context.TypesInfo().Types[source]
	if resultType == nil ||
		expectedType == nil ||
		!types.AssignableTo(resultType, expectedType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if factsOK && sourceFacts.Value != nil {
		return constantvalue.EmitValue(
			context,
			source,
			expectedType,
			sourceFacts.Value,
		)
	}
	argumentType := context.TypesInfo().TypeOf(source.Args[0])
	carrier, ok := complexvalue.Describe(argumentType)
	if !ok ||
		!types.Identical(resultType, carrier.ComponentType()) ||
		!types.AssignableTo(resultType, expectedType) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argument, err := children.Expression(
		context.WithRole(api.RoleCallArgument).WithExpectedType(argumentType),
		source.Args[0],
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		argument.Before(),
		context.Factory().PropertyAccessExpression(
			argument.Value(),
			nil,
			context.Factory().Identifier(member),
			tsgo.NodeFlagsNone,
		),
		argument.Requests(),
	)
}
