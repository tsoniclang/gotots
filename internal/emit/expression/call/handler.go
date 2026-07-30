package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	cooperativecall "github.com/tsoniclang/gotots/internal/emit/concurrency/cooperative"
	builtinexpression "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	conversionexpression "github.com/tsoniclang/gotots/internal/emit/expression/conversion"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	return emit(context, children, source, false)
}

func EmitDiscarded(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
) (api.ExpressionEmission, error) {
	return emit(context, children, source, true)
}

func emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
) (api.ExpressionEmission, error) {
	context, detached := context.TakeDetachedInvocation()
	if target, handled, err := emitGeneric(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, ok, err := conversionexpression.Emit(
		context,
		children,
		source,
	); ok || err != nil {
		if detached && err == nil {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return target, err
	}
	if builtin, ok := builtinexpression.Object(
		context.TypesInfo(),
		source.Fun,
	); ok {
		if detached {
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		if target, handled, err := emitUnsafeBuiltin(
			context,
			children,
			source,
			builtin,
			discarded,
		); handled {
			return target, err
		}
		return builtinexpression.Emit(
			context,
			children,
			source,
			builtin,
			discarded,
		)
	}
	if selector, method, selection, ok := selectedMethod(
		context.TypesInfo(),
		source.Fun,
	); ok {
		return emitMethod(
			context,
			children,
			source,
			selector,
			method,
			selection,
			discarded,
			detached,
		)
	}
	signature, ok := callable.Signature(
		context.TypesInfo().TypeOf(source.Fun),
	)
	if !ok || signature.Recv() != nil {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if identifier, directIdentifier := source.Fun.(*ast.Ident); directIdentifier {
		switch context.TypesInfo().Uses[identifier].(type) {
		case *types.Func, *types.Var:
		default:
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	callee, static, err := emitCallee(context, children, source.Fun, signature)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	guardNil := !static &&
		!callable.StaticallyNonNil(context.TypesInfo(), source.Fun)
	arguments, argumentBefore, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		guardNil || detached,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	targetCallee := callee.Value()
	if _, defined := definedtype.ResolveCallable(
		context.TypesInfo().TypeOf(source.Fun),
	); defined {
		targetCallee = context.Factory().PropertyAccessExpression(
			targetCallee,
			nil,
			context.Factory().Identifier(definedtype.ValueMember),
			tsgo.NodeFlagsNone,
		)
	}
	before := callee.Before()
	if static && len(before) != 0 {
		return api.ExpressionEmission{},
			&api.InvariantError{
				Role:   api.RoleCallCallee,
				Reason: "static callee produced prerequisite statements",
			}
	}
	if guardNil || (!static && (len(argumentBefore) != 0 || detached)) {
		temporaryName, err := context.Names().Temporary(api.TemporaryCallCallee)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(temporaryName),
							nil,
							nil,
							targetCallee,
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		targetCallee = context.Factory().Identifier(temporaryName)
	}
	before = append(before, argumentBefore...)
	var guardRequests []api.RootRequest
	if guardNil {
		if detached {
			targetCallee, guardRequests, err =
				callable.DetachedNilGuard(
					context,
					targetCallee,
					targetCallee,
				)
			if err != nil {
				return api.ExpressionEmission{}, err
			}
		} else {
			guard, requests, guardErr :=
				callable.NilGuard(context, targetCallee)
			if guardErr != nil {
				return api.ExpressionEmission{}, guardErr
			}
			before = append(before, guard)
			guardRequests = requests
		}
	}
	call := context.Factory().CallExpression(
		targetCallee,
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	requests := api.CombineRequests(
		callee.Requests(),
		argumentRequests,
		guardRequests,
	)
	target, err := api.NewExpressionEmission(before, call, requests)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if literal, direct := directFunctionLiteral(source.Fun); direct {
		if detached {
			return cooperativecall.DetachedLiteralCall(
				context,
				source,
				literal,
				target,
			)
		}
		return cooperativecall.LiteralCall(
			context,
			source,
			literal,
			target,
		)
	}
	if provider, direct := calleeObject(
		context.TypesInfo(),
		source.Fun,
	); direct {
		if detached {
			return cooperativecall.DetachedSourceCall(
				context,
				source,
				provider,
				target,
			)
		}
		return cooperativecall.SourceCall(
			context,
			source,
			provider,
			target,
		)
	}
	if detached {
		return cooperativecall.DetachedValueCall(
			context,
			source,
			signature,
			target,
		)
	}
	return cooperativecall.ValueCall(
		context,
		source,
		signature,
		target,
	)
}

func validateResults(
	context api.Context,
	source *ast.CallExpr,
	signature *types.Signature,
	discarded bool,
) error {
	resultCount := 0
	if signature.Results() != nil {
		resultCount = signature.Results().Len()
	}
	if discarded {
		return nil
	}
	switch {
	case resultCount == 0:
		return api.Unsupported(context, api.CategoryExpression, source)
	case resultCount == 1 && context.ExpectedResults() != nil:
		return api.Unsupported(context, api.CategoryExpression, source)
	case resultCount > 1 &&
		(context.ExpectedResults() == nil ||
			!types.Identical(signature.Results(), context.ExpectedResults())):
		return api.Unsupported(context, api.CategoryExpression, source)
	default:
		return nil
	}
}

func calleeObject(info *types.Info, source ast.Expr) (*types.Func, bool) {
	if info == nil {
		return nil, false
	}
	switch source := source.(type) {
	case *ast.Ident:
		object, ok := info.Uses[source].(*types.Func)
		return object, ok
	case *ast.SelectorExpr:
		if info.Selections[source] != nil {
			return nil, false
		}
		qualifier, ok := source.X.(*ast.Ident)
		if !ok {
			return nil, false
		}
		packageName, ok := info.Uses[qualifier].(*types.PkgName)
		if !ok {
			return nil, false
		}
		object, ok := info.Uses[source.Sel].(*types.Func)
		if !ok || object.Pkg() != packageName.Imported() {
			return nil, false
		}
		return object, true
	default:
		return nil, false
	}
}

func emitCallee(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
	signature *types.Signature,
) (api.ExpressionEmission, bool, error) {
	static := false
	if object, ok := calleeObject(context.TypesInfo(), source); ok {
		objectSignature, valid := callable.Signature(object.Type())
		if !valid || !types.Identical(objectSignature, signature) {
			return api.ExpressionEmission{}, false,
				api.Unsupported(
					context.WithRole(api.RoleCallCallee),
					api.CategoryExpression,
					source,
				)
		}
		reference, err := context.Names().Reference(object)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		return api.DirectExpression(
			context.Factory().Identifier(reference.Name()),
			reference.Requests()...,
		), true, nil
	} else if _, ok := directFunctionLiteral(source); ok {
		target, err := children.Expression(
			context.
				WithRole(api.RoleCallCallee).
				WithExpectedType(signature).
				WithStaticallySelectedCallable(),
			source,
		)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		return target, false, nil
	} else if identifier, ok := source.(*ast.Ident); ok {
		variable, valid := context.TypesInfo().Uses[identifier].(*types.Var)
		if !valid {
			return api.ExpressionEmission{}, false,
				api.Unsupported(
					context.WithRole(api.RoleCallCallee),
					api.CategoryExpression,
					source,
				)
		}
		variableSignature, represented := callable.Signature(variable.Type())
		if !represented || !types.Identical(variableSignature, signature) {
			return api.ExpressionEmission{}, false,
				api.Unsupported(
					context.WithRole(api.RoleCallCallee),
					api.CategoryExpression,
					source,
				)
		}
	}
	target, err := children.Expression(
		context.
			WithRole(api.RoleCallCallee).
			WithExpectedType(signature),
		source,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	return target, static, nil
}

func directFunctionLiteral(source ast.Expr) (*ast.FuncLit, bool) {
	for {
		switch selected := source.(type) {
		case *ast.FuncLit:
			return selected, true
		case *ast.ParenExpr:
			source = selected.X
		default:
			return nil, false
		}
	}
}

func emitArguments(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	signature *types.Signature,
	captureAll bool,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	if len(source.Args) == 1 {
		if results, ok := context.TypesInfo().TypeOf(source.Args[0]).(*types.Tuple); ok {
			if signature.Variadic() {
				return emitVariadicMultipleArgument(
					context,
					children,
					source,
					signature,
					results,
					captureAll,
				)
			}
			arguments, before, requests, err := emitMultipleArgument(
				context,
				children,
				source,
				signature,
				results,
			)
			if err != nil || !captureAll {
				return arguments, before, requests, err
			}
			return captureArgumentExpressions(
				context,
				arguments,
				before,
				requests,
			)
		}
	}
	if signature.Variadic() {
		return emitVariadicArguments(
			context,
			children,
			source,
			signature,
			captureAll,
		)
	}
	if signature.Params().Len() != len(source.Args) {
		return nil, nil, nil,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	emissions := make([]api.ExpressionEmission, 0, len(source.Args))
	requiresCapture := false
	for index, argument := range source.Args {
		argumentType := context.TypesInfo().TypeOf(argument)
		if argumentType == nil ||
			!types.AssignableTo(argumentType, signature.Params().At(index).Type()) {
			return nil, nil, nil,
				api.Unsupported(context, api.CategoryExpression, source)
		}
		target, err := children.Expression(
			context.
				WithRole(api.RoleCallArgument).
				WithExpectedType(signature.Params().At(index).Type()),
			argument,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		target, err = context.Values().Copy(
			context.WithRole(api.RoleCallArgument),
			argument,
			signature.Params().At(index).Type(),
			target,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(target.Before()) != 0 {
			requiresCapture = true
		}
		emissions = append(emissions, target)
	}
	if requiresCapture || captureAll {
		return captureArguments(context, children, source, signature, emissions)
	}
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var requests []api.RootRequest
	for _, target := range emissions {
		arguments = append(arguments, target.Value())
		requests = append(requests, target.Requests()...)
	}
	return arguments, nil, requests, nil
}

func captureArgumentExpressions(
	context api.Context,
	expressions []tsgo.Expression,
	before []tsgo.Statement,
	requests []api.RootRequest,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(expressions))
	for _, expression := range expressions {
		temporaryName, err := context.Names().Temporary(
			api.TemporaryCallArgument,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(temporaryName),
							nil,
							nil,
							expression,
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		arguments = append(
			arguments,
			context.Factory().Identifier(temporaryName),
		)
	}
	return arguments, before, requests, nil
}

func captureArguments(
	context api.Context,
	_ api.ChildEmitter,
	_ *ast.CallExpr,
	_ *types.Signature,
	emissions []api.ExpressionEmission,
) ([]tsgo.Expression, []tsgo.Statement, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(emissions))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, emission := range emissions {
		temporaryName, err := context.Names().Temporary(api.TemporaryCallArgument)
		if err != nil {
			return nil, nil, nil, err
		}
		declaration := context.Factory().VariableDeclaration(
			context.Factory().Identifier(temporaryName),
			nil,
			nil,
			emission.Value(),
		)
		before = append(before, emission.Before()...)
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{declaration},
					tsgo.NodeFlagsConst,
				),
			),
		)
		arguments = append(
			arguments,
			context.Factory().Identifier(temporaryName),
		)
		requests = append(requests, emission.Requests()...)
	}
	return arguments, before, requests, nil
}
