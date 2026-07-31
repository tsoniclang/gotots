package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
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
	directOwner, direct := calleeObject(context.TypesInfo(), source.Fun)
	var providerRecovery api.RecoveryCallableReference
	providerRecoverySelected := false
	var err error
	if direct {
		providerRecovery, providerRecoverySelected, err =
			context.Names().RecoveryCallable(directOwner)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	var callee api.ExpressionEmission
	static := false
	if direct && providerRecoverySelected {
		callee = api.DirectExpression(
			providerRecovery.Expression(context.Factory()),
			providerRecovery.Requests()...,
		)
		static = true
	} else {
		callee, static, err = emitCallee(
			context,
			children,
			source.Fun,
			signature,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetCallee := callee.Value()
	before := callee.Before()
	var requests []api.RootRequest
	var contractRequests []api.RootRequest
	cooperative := false
	literal, directLiteral := directFunctionLiteral(source.Fun)
	switch {
	case static:
		owner := directOwner
		if !direct || owner == nil {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   api.RoleCallCallee,
				Reason: "static deferred callee has no exact function owner",
			}
		}
		if providerRecoverySelected {
			cooperative = providerRecovery.Cooperative()
		} else {
			control, controlErr := api.NewDirectCallableControlRequest(
				owner.Origin(),
				api.CallableControlRecovery,
			)
			if controlErr != nil {
				return api.ExpressionEmission{}, controlErr
			}
			requests = append(requests, control)
			cooperative, contractRequests, err =
				cooperativecall.SourceContract(context, owner)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		}
	case directLiteral:
		name, err := context.Names().Temporary(api.TemporaryCallCallee)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(
			before,
			constantDeclaration(
				context,
				name,
				nil,
				targetCallee,
			),
		)
		targetCallee = context.Factory().Identifier(name)
		control, err := context.FunctionLiteralControlRequest(
			literal,
			api.CallableControlRecovery,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		requests = append(requests, control)
		cooperative, contractRequests, err =
			cooperativecall.LiteralContract(context, literal)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	default:
		sourceType := context.TypesInfo().TypeOf(source.Fun)
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleCallCallee),
			source.Fun,
			sourceType,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if _, defined := definedtype.ResolveCallable(sourceType); defined {
			targetCallee = context.Factory().PropertyAccessExpression(
				targetCallee,
				nil,
				context.Factory().Identifier(definedtype.ValueMember),
				tsgo.NodeFlagsNone,
			)
			targetType, err = callable.EmitType(
				context.WithRole(api.RoleCallCallee),
				children,
				source.Fun,
				signature,
			)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
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
		cooperative, contractRequests, err =
			cooperativecall.ValueContract(context, signature)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	requests = append(requests, contractRequests...)
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
		cooperative,
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
			false,
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
	targetCooperative bool,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	recovery, recoveryRequests, err :=
		callable.RecoveryAuthorityParameter(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if targetCooperative {
		request, requestErr := context.CooperativeRequest()
		if requestErr != nil {
			return api.ExpressionEmission{}, requestErr
		}
		requests = append(requests, request)
		call = context.Factory().AwaitExpression(call)
	}
	cooperative := targetCooperative || context.IsCooperative()
	body := append(invocationBefore, context.Factory().ExpressionStatement(call))
	var modifiers []tsgo.ModifierLike
	var resultType tsgo.TypeNode = context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	if cooperative {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			[]tsgo.ParameterDeclaration{recovery},
			resultType,
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
	var modifiers []tsgo.ModifierLike
	var resultType tsgo.TypeNode = context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	if context.IsCooperative() {
		modifiers = []tsgo.ModifierLike{context.Factory().AsyncKeyword()}
		resultType = callable.PromiseResult(context.Factory(), resultType)
	}
	return api.NewExpressionEmission(
		nil,
		context.Factory().ArrowFunction(
			modifiers,
			nil,
			[]tsgo.ParameterDeclaration{recovery},
			resultType,
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
