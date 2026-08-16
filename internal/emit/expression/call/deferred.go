package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	builtinexpression "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
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
	literal, directLiteral := directFunctionLiteral(source.Fun)
	var providerRecovery api.RecoveryCallableReference
	providerRecoverySelected := false
	sourceRecovery := false
	var recoveryRequests []api.RootRequest
	var err error
	if direct {
		providerRecovery, providerRecoverySelected, err =
			context.Names().RecoveryCallable(directOwner)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if !providerRecoverySelected {
			facet, facetErr := api.NewSourceCallableFacet(directOwner)
			if facetErr != nil {
				return api.ExpressionEmission{}, facetErr
			}
			observation, observationErr :=
				context.ObserveRecoveryCallable(facet)
			if observationErr != nil {
				return api.ExpressionEmission{}, observationErr
			}
			sourceRecovery = observation.Recovery()
			recoveryRequests = observation.Requests()
		}
	}
	var callee api.ExpressionEmission
	static := false
	providerBoundary := false
	if directLiteral {
		callee, err = children.Expression(
			context.
				WithRole(api.RoleCallCallee).
				WithExpectedType(signature).
				WithStaticallySelectedCallable().
				WithDeferredCallableSelection(),
			source.Fun,
		)
	} else if direct {
		if providerRecoverySelected {
			callee = api.DirectExpression(
				providerRecovery.Expression(context.Factory()),
				providerRecovery.Requests()...,
			)
			providerBoundary = providerRecovery.ProviderBoundary()
		} else if sourceRecovery {
			deferredReference, referenceErr :=
				context.Names().DeferredCallable(directOwner, "")
			if referenceErr != nil {
				return api.ExpressionEmission{}, referenceErr
			}
			callee = api.DirectExpression(
				deferredReference.Expression(context.Factory()),
				deferredReference.Requests()...,
			)
		} else {
			reference, referenceErr := context.Names().Reference(directOwner)
			if referenceErr != nil {
				return api.ExpressionEmission{}, referenceErr
			}
			callee = api.DirectExpression(
				reference.Expression(context.Factory()),
				reference.Requests()...,
			)
			providerBoundary = reference.ProviderBoundary()
		}
		static = true
	} else {
		callee, static, providerBoundary, err = emitCallee(
			context,
			children,
			source.Fun,
			signature,
		)
	}
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	sourceType := context.TypesInfo().TypeOf(source.Fun)
	if model, defined := definedtype.ResolveCallable(sourceType); defined {
		providerCarrier, carrierErr := model.ProviderCarrier(
			context.WithRole(api.RoleCallCallee),
		)
		if carrierErr != nil {
			return api.ExpressionEmission{}, carrierErr
		}
		providerBoundary = providerBoundary || providerCarrier
		callee, err = model.Project(
			context.WithRole(api.RoleCallCallee),
			callee,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	targetCallee := callee.Value()
	before := callee.Before()
	requests := recoveryRequests
	var contractRequests []api.RootRequest
	cooperative := false
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
		cooperative, contractRequests, err =
			cooperativecall.LiteralContract(context, literal)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	default:
		exactSynchronous, exactRequests, selectionErr :=
			cooperativecall.ExactSynchronousValue(context, source.Fun)
		if selectionErr != nil {
			return api.ExpressionEmission{}, selectionErr
		}
		var targetType api.TypeEmission
		if exactSynchronous {
			targetType, err = callable.EmitSynchronousType(
				context.WithRole(api.RoleCallCallee),
				children,
				source.Fun,
				signature,
			)
		} else {
			targetType, err = children.RepresentedType(
				context.WithRole(api.RoleCallCallee),
				source.Fun,
				sourceType,
			)
		}
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if _, defined := definedtype.ResolveCallable(sourceType); !exactSynchronous &&
			defined &&
			!providerBoundary {
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
		var valueRequests []api.RootRequest
		cooperative, valueRequests, err =
			cooperativecall.ValueContract(context, signature)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		contractRequests = api.CombineRequests(
			exactRequests,
			valueRequests,
		)
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
	if providerBoundary {
		var providerBefore []tsgo.Statement
		var providerRequests []api.RootRequest
		arguments, providerBefore, providerRequests, err =
			providerboundary.ToProviderArguments(
				context,
				children,
				signature.Params(),
				arguments,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		argumentBefore = append(argumentBefore, providerBefore...)
		argumentRequests = api.CombineRequests(
			argumentRequests,
			providerRequests,
		)
	}
	before = append(before, argumentBefore...)
	literalRecovery := directLiteral &&
		context.CallableControlFor(literal).Recovery()
	var call tsgo.Expression
	if providerRecoverySelected {
		arguments = append(
			arguments,
			context.Factory().Identifier(callable.RecoveryAuthorityName),
		)
		call = context.Factory().CallExpression(
			targetCallee,
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		)
	} else if sourceRecovery || literalRecovery {
		arguments = append(
			[]tsgo.Expression{
				context.Factory().Identifier(callable.RecoveryAuthorityName),
			},
			arguments...,
		)
		call = context.Factory().CallExpression(
			targetCallee,
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		)
	} else if !static && !directLiteral {
		registry, registryErr := deferredregistry.Reference(
			context,
			source,
			signature,
		)
		if registryErr != nil {
			return api.ExpressionEmission{}, registryErr
		}
		deferredName, nameErr := context.Names().Temporary(
			api.TemporaryDeferredCall,
		)
		if nameErr != nil {
			return api.ExpressionEmission{}, nameErr
		}
		before = append(
			before,
			constantDeclaration(
				context,
				deferredName,
				nil,
				context.Factory().CallExpression(
					context.Factory().PropertyAccessExpression(
						registry.Expression(context.Factory()),
						nil,
						context.Factory().Identifier(
							api.DeferredRegistryResolveName,
						),
						tsgo.NodeFlagsNone,
					),
					nil,
					nil,
					[]tsgo.Expression{targetCallee},
					tsgo.NodeFlagsNone,
				),
			),
		)
		guarded, guardRequests, guardErr := callable.NilGuardExpression(
			context,
			targetCallee,
		)
		if guardErr != nil {
			return api.ExpressionEmission{}, guardErr
		}
		ordinaryCall := context.Factory().CallExpression(
			guarded,
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		)
		deferredCall := context.Factory().CallExpression(
			context.Factory().Identifier(deferredName),
			nil,
			nil,
			append(
				[]tsgo.Expression{
					context.Factory().Identifier(
						callable.RecoveryAuthorityName,
					),
				},
				arguments...,
			),
			tsgo.NodeFlagsNone,
		)
		call = context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(deferredName),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				context.Factory().Identifier("undefined"),
			),
			context.Factory().QuestionToken(),
			ordinaryCall,
			context.Factory().ColonToken(),
			deferredCall,
		)
		requests = append(
			requests,
			api.CombineRequests(
				registry.Requests(),
				guardRequests,
			)...,
		)
	} else {
		call = context.Factory().CallExpression(
			targetCallee,
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		)
	}
	return deferredInvocation(
		context,
		before,
		nil,
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
	body := append(invocationBefore, context.Factory().ExpressionStatement(call))
	var modifiers []tsgo.ModifierLike
	var resultType tsgo.TypeNode = context.Factory().KeywordTypeNode(
		tsgo.KeywordTypeSyntaxKindVoidKeyword,
	)
	if targetCooperative {
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
