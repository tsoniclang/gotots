package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	builtinexpression "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitDeferred(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	if source == nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if target, handled, err := emitDeferredGeneric(
		context,
		children,
		source,
	); handled {
		return target, err
	}
	if builtin, ok := builtinexpression.Object(
		context.TypesInfo(),
		source.Fun,
	); ok {
		return emitDeferredBuiltin(
			context,
			children,
			source,
			builtin,
		)
	}
	if selector, method, selection, ok := selectedMethod(
		context.TypesInfo(),
		source.Fun,
	); ok {
		return emitDeferredMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
		)
	}
	signature, ok := callable.Signature(
		context.TypesInfo().TypeOf(source.Fun),
	)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if err := validateResults(context, source, signature, true); err != nil {
		return api.ExpressionEmission{}, err
	}
	callee, static, err := emitCallee(
		context,
		children,
		source.Fun,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetCallee := callee.Value()
	before := callee.Before()
	var requests []api.RootRequest
	if static {
		owner, direct := calleeObject(context.TypesInfo(), source.Fun)
		if !direct {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   api.RoleCallCallee,
				Reason: "static deferred callee has no exact function owner",
			}
		}
		control, err := api.NewDirectCallableControlRequest(
			owner.Origin(),
			api.CallableControlRecovery,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		requests = append(requests, control)
	} else {
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleCallCallee),
			source.Fun,
			context.TypesInfo().TypeOf(source.Fun),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		name, err := context.Names().Temporary(api.TemporaryCallCallee)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(
			before,
			constantDeclaration(
				context,
				name,
				targetType.Value(),
				targetCallee,
			),
		)
		targetCallee = context.Factory().Identifier(name)
		requests = append(requests, targetType.Requests()...)
	}
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		true,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	before = append(before, argumentBefore...)
	var invocationBefore []tsgo.Statement
	if !static &&
		!callable.StaticallyNonNil(context.TypesInfo(), source.Fun) {
		guard, guardRequests, err := callable.NilGuard(context, targetCallee)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		invocationBefore = append(invocationBefore, guard)
		requests = append(requests, guardRequests...)
		if _, ok := definedtype.ResolveCallable(
			context.TypesInfo().TypeOf(source.Fun),
		); ok {
			targetCallee = context.Factory().PropertyAccessExpression(
				targetCallee,
				nil,
				context.Factory().Identifier(definedtype.ValueMember),
				tsgo.NodeFlagsNone,
			)
		}
	}
	arguments = append(
		arguments,
		context.Factory().Identifier(callable.RecoveryAuthorityName),
	)
	call := context.Factory().CallExpression(
		targetCallee,
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	return deferredInvocation(
		context,
		before,
		invocationBefore,
		call,
		api.CombineRequests(
			callee.Requests(),
			argumentRequests,
			requests,
		),
	)
}

func emitDeferredBuiltin(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	builtin *types.Builtin,
) (api.ExpressionEmission, error) {
	if target, handled, err := builtinexpression.EmitDeferred(
		context,
		children,
		source,
		builtin,
	); handled {
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return deferredInvocation(
			context,
			target.Before(),
			nil,
			target.Value(),
			target.Requests(),
		)
	}
	if types.Object(builtin) == types.Universe.Lookup("recover") &&
		source != nil &&
		len(source.Args) == 0 &&
		!source.Ellipsis.IsValid() {
		return deferredNoop(context)
	}
	return api.ExpressionEmission{},
		api.Unsupported(context, api.CategoryStatement, source)
}

func deferredInvocation(
	context api.Context,
	before []tsgo.Statement,
	invocationBefore []tsgo.Statement,
	call tsgo.Expression,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	body := append(invocationBefore, context.Factory().ExpressionStatement(call))
	return api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{recovery},
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().Block(body, true),
		),
		api.CombineRequests(requests, recoveryRequests),
	)
}

func deferredNoop(
	context api.Context,
) (api.ExpressionEmission, error) {
	recovery, requests, err := callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		nil,
		context.Factory().ArrowFunction(
			nil,
			nil,
			[]tsgo.ParameterDeclaration{recovery},
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			context.Factory().EqualsGreaterThanToken(),
			context.Factory().Block(nil, true),
		),
		requests,
	)
}

func constantDeclaration(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{
				context.Factory().VariableDeclaration(
					context.Factory().Identifier(name),
					nil,
					targetType,
					value,
				),
			},
			tsgo.NodeFlagsConst,
		),
	)
}
