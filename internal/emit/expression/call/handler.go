package call

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	builtinexpression "github.com/tsoniclang/gotots/internal/emit/expression/builtin"
	conversionexpression "github.com/tsoniclang/gotots/internal/emit/expression/conversion"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
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
	if target, handled, err := emitReflectionTypeOf(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitReflectionMapOf(
		context,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitReflectionMakeSlice(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitReflectionValueOf(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitBinaryCodec(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitReflectionDeepEqual(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitReflectionTypeAssert(
		context,
		children,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
	if target, handled, err := emitReflectionTypeFor(
		context,
		source,
		discarded,
		detached,
	); handled {
		return target, err
	}
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
		switch context.TypesInfo().UseOf(identifier).(type) {
		case *types.Func, *types.Var:
		default:
			return api.ExpressionEmission{},
				api.Unsupported(context, api.CategoryExpression, source)
		}
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, err
	}
	var profileRequests []api.RootRequest
	directFunction, direct := calleeObject(
		context.TypesInfo(),
		source.Fun,
	)
	if direct {
		target, selected, requests, err := emitProviderProfileFunction(
			context,
			children,
			source,
			directFunction,
			signature,
			discarded,
			detached,
		)
		if selected || err != nil {
			return target, err
		}
		profileRequests = requests
	}
	callee, static, providerBoundary, err := emitCallee(
		context,
		children,
		source.Fun,
		signature,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if model, defined := definedtype.ResolveCallable(
		context.TypesInfo().TypeOf(source.Fun),
	); defined {
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
	targetCallee := callee.Value()
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
		targetCallee, guardRequests, err =
			callable.NilGuardExpression(
				context,
				targetCallee,
			)
		if err != nil {
			return api.ExpressionEmission{}, err
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
		profileRequests,
	)
	target, err := api.NewExpressionEmission(before, call, requests)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if discarded || !providerBoundary {
		return target, nil
	}
	return providerboundary.FromProviderResults(
		context,
		children,
		nil,
		"",
		signature.Results(),
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

func calleeObject(info api.TypeInfoView, source ast.Expr) (*types.Func, bool) {
	if !info.Valid() {
		return nil, false
	}
	switch source := source.(type) {
	case *ast.Ident:
		object, ok := info.UseOf(source).(*types.Func)
		return object, ok
	case *ast.SelectorExpr:
		if info.SelectionOf(source) != nil {
			return nil, false
		}
		qualifier, ok := source.X.(*ast.Ident)
		if !ok {
			return nil, false
		}
		packageName, ok := info.UseOf(qualifier).(*types.PkgName)
		if !ok {
			return nil, false
		}
		object, ok := info.UseOf(source.Sel).(*types.Func)
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
) (api.ExpressionEmission, bool, bool, error) {
	static := false
	if object, ok := calleeObject(context.TypesInfo(), source); ok {
		objectSignature, valid := callable.Signature(object.Type())
		if !valid || !types.Identical(objectSignature, signature) {
			return api.ExpressionEmission{}, false, false,
				api.Unsupported(
					context.WithRole(api.RoleCallCallee),
					api.CategoryExpression,
					source,
				)
		}
		reference, err := context.Names().Reference(object)
		if err != nil {
			return api.ExpressionEmission{}, false, false, err
		}
		if reference.ProviderBoundary() {
			if err := providerboundary.RequireProviderCallable(
				context,
				object,
			); err != nil {
				return api.ExpressionEmission{}, false, false, err
			}
		}
		return api.DirectExpression(
			reference.Expression(context.Factory()),
			reference.Requests()...,
		), true, reference.ProviderBoundary(), nil
	} else if _, ok := directFunctionLiteral(source); ok {
		target, err := children.Expression(
			context.
				WithRole(api.RoleCallCallee).
				WithExpectedType(signature).
				WithStaticallySelectedCallable(),
			source,
		)
		if err != nil {
			return api.ExpressionEmission{}, false, false, err
		}
		return target, false, false, nil
	} else if identifier, ok := source.(*ast.Ident); ok {
		variable, valid := context.TypesInfo().UseOf(identifier).(*types.Var)
		if !valid {
			return api.ExpressionEmission{}, false, false,
				api.Unsupported(
					context.WithRole(api.RoleCallCallee),
					api.CategoryExpression,
					source,
				)
		}
		variableSignature, represented := callable.Signature(variable.Type())
		if !represented || !types.Identical(variableSignature, signature) {
			return api.ExpressionEmission{}, false, false,
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
		return api.ExpressionEmission{}, false, false, err
	}
	return target, static, false, nil
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
