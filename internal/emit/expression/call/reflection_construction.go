package call

import (
	"go/ast"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

// emitReflectionMakeSlice intercepts reflect.MakeSlice and
// reflect.MakeMap to demand the generated value facet of the constructed
// type whenever the Type argument carries static composition evidence (a
// direct reflect.TypeOf, reflect.TypeFor, or reflect.MapOf descriptor),
// then delegates the call to the ordinary certified provider-profile
// emission. A dynamically flowing descriptor keeps the certified provider
// boundary and fails loudly at the typed provider check when its facet is
// absent.
func emitReflectionMakeSlice(
	context api.Context,
	children api.ChildEmitter,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, error) {
	owner, ok := calleeObject(context.TypesInfo(), source.Fun)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var arity int
	pointerDemand := false
	switch contract.Identity() {
	case reflectMakeSliceIdentity:
		arity = 3
	case reflectMakeMapIdentity, reflectZeroIdentity:
		arity = 1
	case reflectNewIdentity:
		arity = 1
		pointerDemand = true
	default:
		return api.ExpressionEmission{}, false, nil
	}
	if detached {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok || signature.Results() == nil || signature.Results().Len() != 1 ||
		signature.Params() == nil || signature.Params().Len() != arity ||
		len(source.Args) != arity {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var demand []api.RootRequest
	if composedType, evident := reflectionCompositionType(
		context,
		source.Args[0],
	); evident {
		if pointerDemand {
			composedType = types.NewPointer(composedType)
		}
		reflectTypeObject, typeOK := owner.Pkg().
			Scope().
			Lookup("Type").(*types.TypeName)
		if !typeOK {
			return api.ExpressionEmission{}, true, &api.ContextError{
				Reason: "reflect package has no Type declaration",
			}
		}
		names, namesOK := context.Names().(api.ReflectionNames)
		if !namesOK {
			return api.ExpressionEmission{}, true, &api.ContextError{
				Reason: "reflection names are unavailable",
			}
		}
		metadata, metadataErr := names.ReflectionValueType(
			composedType,
			reflectTypeObject,
		)
		if metadataErr != nil {
			return api.ExpressionEmission{}, true, metadataErr
		}
		demand = metadata.Requests()
	}
	if arity == 3 {
		// MakeSlice carries scalar parameters, so its call must keep the
		// certified provider-profile boundary conversions.
		target, selected, _, err := emitProviderProfileFunction(
			context,
			children,
			source,
			owner,
			signature,
			discarded,
			detached,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		if !selected {
			return api.ExpressionEmission{}, false, nil
		}
		merged, err := api.NewExpressionEmission(
			target.Before(),
			target.Value(),
			api.CombineRequests(target.Requests(), demand),
		)
		return merged, true, err
	}
	// Type-only constructors have no scalar parameters: the ordinary
	// provider call needs no callable-profile bridging.
	callee, err := context.Names().Reference(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	arguments, before, argumentRequests, err := emitArguments(
		context,
		children,
		source,
		signature,
		false,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	emission, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			callee.Expression(context.Factory()),
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(callee.Requests(), argumentRequests, demand),
	)
	return emission, true, err
}

// reflectionCompositionType recovers the exact Go type flowing through a
// direct reflect.TypeOf or reflect.TypeFor composition. Any other Type
// expression yields no static evidence.
func reflectionCompositionType(
	context api.Context,
	expression ast.Expr,
) (types.Type, bool) {
	call, ok := ast.Unparen(expression).(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	if owner, ownerOK := calleeObject(
		context.TypesInfo(),
		call.Fun,
	); ownerOK {
		contract, err := environmentcontract.Describe(owner)
		if err != nil {
			return nil, false
		}
		switch contract.Identity() {
		case reflectTypeOfIdentity:
			if len(call.Args) != 1 {
				return nil, false
			}
			argumentType := context.TypesInfo().TypeOf(call.Args[0])
			if argumentType != nil &&
				!api.ContainsGenericTypeParameter(argumentType) {
				return argumentType, true
			}
		case reflectMapOfIdentity:
			if len(call.Args) != 2 {
				return nil, false
			}
			keyType, keyOK := reflectionCompositionType(
				context,
				call.Args[0],
			)
			elementType, elementOK := reflectionCompositionType(
				context,
				call.Args[1],
			)
			if keyOK && elementOK {
				return types.NewMap(keyType, elementType), true
			}
		case reflectPointerToIdentity:
			if len(call.Args) != 1 {
				return nil, false
			}
			if elementType, elementOK := reflectionCompositionType(
				context,
				call.Args[0],
			); elementOK {
				return types.NewPointer(elementType), true
			}
		case reflectSliceOfIdentity:
			if len(call.Args) != 1 {
				return nil, false
			}
			if elementType, elementOK := reflectionCompositionType(
				context,
				call.Args[0],
			); elementOK {
				return types.NewSlice(elementType), true
			}
		}
		return nil, false
	}
	if owner, instance, instanceOK := genericFunctionInstance(
		context,
		call,
	); instanceOK {
		contract, err := environmentcontract.Describe(owner)
		if err == nil && contract.Identity() == reflectTypeForIdentity &&
			instance.TypeArgs.Len() == 1 &&
			!instance.TypeArgs.ContainsGenericTypeParameter() {
			return instance.TypeArgs.At(0), true
		}
	}
	return nil, false
}

// emitReflectionMapOf intercepts the descriptor combinators MapOf,
// PointerTo, and SliceOf when every Type argument is an effect-free
// descriptor composition: the call collapses to the canonical combined
// descriptor reference on the generated-facet route, so no provider
// binding participates. Argument expressions with observable evaluation
// keep the ordinary provider boundary.
func emitReflectionMapOf(
	context api.Context,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, error) {
	owner, ok := calleeObject(context.TypesInfo(), source.Fun)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	var arity int
	switch contract.Identity() {
	case reflectMapOfIdentity:
		arity = 2
	case reflectPointerToIdentity, reflectSliceOfIdentity:
		arity = 1
	default:
		return api.ExpressionEmission{}, false, nil
	}
	if discarded || detached || len(source.Args) != arity {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	for _, argument := range source.Args {
		if !effectFreeCompositionOperand(argument) {
			return api.ExpressionEmission{}, false, nil
		}
	}
	combined, combinedOK := reflectionCompositionType(context, source)
	if !combinedOK {
		return api.ExpressionEmission{}, false, nil
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok || signature.Results() == nil || signature.Results().Len() != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	reflectionType, ok := types.Unalias(
		signature.Results().At(0).Type(),
	).(*types.Named)
	if !ok || reflectionType.Obj() == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	names, ok := context.Names().(api.ReflectionNames)
	if !ok {
		return api.ExpressionEmission{}, true, &api.ContextError{
			Reason: "reflection names are unavailable",
		}
	}
	observer, ok := context.Names().(environmentcontract.ImplementationObserver)
	if !ok {
		return api.ExpressionEmission{}, true, &api.ContextError{
			Reason: "environment implementation observer is unavailable",
		}
	}
	if err := observer.ObserveEnvironmentImplementation(
		owner,
		environmentcontract.UseDemandCallable,
		environmentcontract.RouteGeneratedFacet,
	); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := names.ReflectionType(
		combined,
		reflectionType.Obj(),
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	return api.DirectExpression(
		reference.Expression(context.Factory()),
		reference.Requests()...,
	), true, nil
}

// effectFreeCompositionOperand admits descriptor compositions whose
// evaluation is observably pure: TypeFor takes no arguments, and TypeOf
// or nested MapOf operands must bottom out in identifiers and literals.
func effectFreeCompositionOperand(expression ast.Expr) bool {
	call, ok := ast.Unparen(expression).(*ast.CallExpr)
	if !ok {
		return false
	}
	for _, argument := range call.Args {
		switch operand := ast.Unparen(argument).(type) {
		case *ast.Ident, *ast.BasicLit:
		case *ast.CallExpr:
			if !effectFreeCompositionOperand(operand) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
