package call

import (
	"go/ast"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit/api"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const reflectTypeForIdentity = "reflect|kind=4|receiver=|name=TypeFor"
const reflectValueOfIdentity = "reflect|kind=4|receiver=|name=ValueOf"
const reflectTypeOfIdentity = "reflect|kind=4|receiver=|name=TypeOf"
const reflectMakeSliceIdentity = "reflect|kind=4|receiver=|name=MakeSlice"
const reflectMakeMapIdentity = "reflect|kind=4|receiver=|name=MakeMap"
const reflectMapOfIdentity = "reflect|kind=4|receiver=|name=MapOf"

func emitReflectionTypeOf(
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
	if contract.Identity() != reflectTypeOfIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if detached {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok || signature.Results() == nil || signature.Results().Len() != 1 ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
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
	if signature.Params() == nil || signature.Params().Len() != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	argumentType := context.TypesInfo().TypeOf(source.Args[0])
	if argumentType == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if basic, ok := types.Unalias(argumentType).(*types.Basic); ok &&
		basic.Kind() == types.UntypedNil {
		zero, zeroErr := context.Values().Zero(
			context,
			source,
			signature.Results().At(0).Type(),
		)
		return zero, true, zeroErr
	}
	if api.ContainsGenericTypeParameter(argumentType) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operations, err := names.ReflectionTypeOf(
		argumentType,
		reflectionType.Obj(),
	)
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
	typeOf, err := operations.MemberExpression(context.Factory(), "$typeOf")
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	emission, err := api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			typeOf,
			nil,
			nil,
			arguments,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(
			operations.Requests(),
			argumentRequests,
		),
	)
	return emission, true, err
}

func emitReflectionTypeFor(
	context api.Context,
	source *ast.CallExpr,
	discarded bool,
	detached bool,
) (api.ExpressionEmission, bool, error) {
	owner, instance, ok := genericFunctionInstance(context, source)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	contract, err := environmentcontract.Describe(owner)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if contract.Identity() != reflectTypeForIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if discarded || detached || len(source.Args) != 0 ||
		instance.TypeArgs.Len() != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	signature, ok := instance.Type.(*types.Signature)
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
	if instance.TypeArgs.ContainsGenericTypeParameter() {
		if err := observer.ObserveEnvironmentImplementation(
			owner,
			environmentcontract.UseDemandCallable,
			environmentcontract.RouteGeneratedFacet,
		); err != nil {
			return api.ExpressionEmission{}, true, err
		}
		witnessType := types.NewPointer(instance.TypeArgs.At(0))
		witness, err := context.Values().Zero(context, source, witnessType)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		emission, err := genericoperation.Call(
			context,
			source,
			api.GenericOperationReflectionType,
			[]types.Type{witnessType},
			[]types.Type{signature.Results().At(0).Type()},
			[]api.ExpressionEmission{witness},
		)
		return emission, true, err
	}
	if err := observer.ObserveEnvironmentImplementation(
		owner,
		environmentcontract.UseDemandCallable,
		environmentcontract.RouteGeneratedFacet,
	); err != nil {
		return api.ExpressionEmission{}, true, err
	}
	reference, err := names.ReflectionType(
		instance.TypeArgs.At(0),
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

// emitReflectionValueOf intercepts reflect.ValueOf to demand the canonical
// descriptor plus the generated value-operation facet for the exact operand
// type, then emits the ordinary provider call unchanged: the source-facing
// call keeps its one Go argument and its certified provider binding.
func emitReflectionValueOf(
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
	if contract.Identity() != reflectValueOfIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if detached {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	signature, ok := owner.Type().(*types.Signature)
	if !ok || signature.Results() == nil || signature.Results().Len() != 1 ||
		signature.Params() == nil || signature.Params().Len() != 1 ||
		len(source.Args) != 1 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if err := validateResults(context, source, signature, discarded); err != nil {
		return api.ExpressionEmission{}, true, err
	}
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
	requests := api.CombineRequests(callee.Requests(), argumentRequests)
	argumentType := context.TypesInfo().TypeOf(source.Args[0])
	if argumentType == nil {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	untypedNil := false
	if basic, basicOK := types.Unalias(argumentType).(*types.Basic); basicOK &&
		basic.Kind() == types.UntypedNil {
		untypedNil = true
	}
	if !untypedNil {
		if api.ContainsGenericTypeParameter(argumentType) {
			return api.ExpressionEmission{}, true,
				api.Unsupported(context, api.CategoryExpression, source)
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
		metadata, metadataErr := names.ReflectionValueOf(
			argumentType,
			reflectTypeObject,
		)
		if metadataErr != nil {
			return api.ExpressionEmission{}, true, metadataErr
		}
		requests = api.CombineRequests(requests, metadata.Requests())
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
		requests,
	)
	return emission, true, err
}

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
	switch contract.Identity() {
	case reflectMakeSliceIdentity:
		arity = 3
	case reflectMakeMapIdentity:
		arity = 1
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
	if sliceType, evident := reflectionCompositionType(
		context,
		source.Args[0],
	); evident {
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
			sliceType,
			reflectTypeObject,
		)
		if metadataErr != nil {
			return api.ExpressionEmission{}, true, metadataErr
		}
		demand = metadata.Requests()
	}
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

// emitReflectionMapOf intercepts reflect.MapOf when both Type arguments
// are effect-free descriptor compositions: the call collapses to the
// canonical map descriptor reference on the generated-facet route, so no
// provider binding participates. Argument expressions with observable
// evaluation keep the ordinary provider boundary.
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
	if contract.Identity() != reflectMapOfIdentity {
		return api.ExpressionEmission{}, false, nil
	}
	if discarded || detached || len(source.Args) != 2 {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	if !effectFreeCompositionOperand(source.Args[0]) ||
		!effectFreeCompositionOperand(source.Args[1]) {
		return api.ExpressionEmission{}, false, nil
	}
	keyType, keyOK := reflectionCompositionType(context, source.Args[0])
	elementType, elementOK := reflectionCompositionType(
		context,
		source.Args[1],
	)
	if !keyOK || !elementOK {
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
		types.NewMap(keyType, elementType),
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
