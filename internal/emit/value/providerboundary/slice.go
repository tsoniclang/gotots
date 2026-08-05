package providerboundary

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type sliceBoundaryDirection uint8

const (
	sliceBoundaryInvalid sliceBoundaryDirection = iota
	sliceBoundaryFromProvider
	sliceBoundaryToProvider
)

func fromProviderSlice(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	return projectProviderSlice(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		sourceType,
		value,
		sliceBoundaryFromProvider,
	)
}

func toProviderSlice(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
) (api.ExpressionEmission, bool, bool, error) {
	return projectProviderSlice(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		sourceType,
		value,
		sliceBoundaryToProvider,
	)
}

func projectProviderSlice(
	context api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	sourceType types.Type,
	value api.ExpressionEmission,
	direction sliceBoundaryDirection,
) (api.ExpressionEmission, bool, bool, error) {
	slice, ok := types.Unalias(sourceType).(*types.Slice)
	if !ok {
		return value, false, false, nil
	}
	if direction != sliceBoundaryFromProvider &&
		direction != sliceBoundaryToProvider {
		return api.ExpressionEmission{}, true, false, boundaryInvariant(
			context,
			"provider slice boundary direction is invalid",
		)
	}
	providerContext, err := providerRepresentationContext(context, profile)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	element := slice.Elem()
	productType, err := context.ContainerStorage().ContainerStorageType(
		context.WithRole(api.RoleSliceElementType),
		nil,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	providerType, err := providerContext.ContainerStorage().ContainerStorageType(
		providerContext.WithRole(api.RoleSliceElementType),
		nil,
		element,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	fromProvider, fromChanged, err := providerSliceElementConversion(
		context,
		providerContext,
		children,
		owner,
		ownerBridge,
		profile,
		element,
		"$providerElement",
		sliceBoundaryFromProvider,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	toProvider, toChanged, err := providerSliceElementConversion(
		context,
		providerContext,
		children,
		owner,
		ownerBridge,
		profile,
		element,
		"$productElement",
		sliceBoundaryToProvider,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	storageChanged := context.Values().RequiresStorageProjection(context, element) !=
		providerContext.Values().RequiresStorageProjection(providerContext, element)
	if !fromChanged && !toChanged && !storageChanged {
		return value, true, false, nil
	}
	productZero, err := sliceStorageZero(context, element)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	providerZero, err := sliceStorageZero(providerContext, element)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	projection, err := context.Names().Runtime(
		api.RuntimeSliceProjection,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, false, err
	}
	fromArrow := conversionArrow(
		context,
		"$providerElement",
		providerType.Value(),
		productType.Value(),
		fromProvider,
	)
	toArrow := conversionArrow(
		context,
		"$productElement",
		productType.Value(),
		providerType.Value(),
		toProvider,
	)
	sourceElement := providerType.Value()
	targetElement := productType.Value()
	fromSource := fromArrow
	toSource := toArrow
	sourceZero := providerZero
	targetZero := productZero
	if direction == sliceBoundaryToProvider {
		sourceElement = productType.Value()
		targetElement = providerType.Value()
		fromSource = toArrow
		toSource = fromArrow
		sourceZero = productZero
		targetZero = providerZero
	}
	target, err := api.NewExpressionEmission(
		append(append(value.Before(), sourceZero.Before()...), targetZero.Before()...),
		context.Factory().NewExpression(
			context.Factory().Identifier(projection.Name()),
			[]tsgo.TypeNode{sourceElement, targetElement},
			[]tsgo.Expression{
				value.Value(),
				fromSource,
				toSource,
				sourceZero.Value(),
				targetZero.Value(),
			},
		),
		api.CombineRequests(
			value.Requests(),
			productType.Requests(),
			providerType.Requests(),
			fromProvider.Requests(),
			toProvider.Requests(),
			productZero.Requests(),
			providerZero.Requests(),
			projection.Requests(),
		),
	)
	return target, true, err == nil, err
}

func providerSliceElementConversion(
	context api.Context,
	providerContext api.Context,
	children api.ChildEmitter,
	owner *types.Named,
	ownerBridge string,
	profile []gostdlib.ProviderCallableProfileInterface,
	element types.Type,
	parameter string,
	direction sliceBoundaryDirection,
) (api.ExpressionEmission, bool, error) {
	value := api.DirectExpression(context.Factory().Identifier(parameter))
	var err error
	if direction == sliceBoundaryFromProvider {
		value, err = providerContext.ContainerStorage().FromContainerStorage(
			providerContext.WithRole(api.RoleSliceElement),
			nil,
			element,
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		var changed bool
		value, changed, err = fromProviderValueSelected(
			context,
			children,
			owner,
			ownerBridge,
			profile,
			element,
			value,
		)
		if err != nil {
			return api.ExpressionEmission{}, false, err
		}
		value, err = context.ContainerStorage().ToContainerStorage(
			context.WithRole(api.RoleSliceElement),
			nil,
			element,
			value,
		)
		return value, changed, err
	}
	value, err = context.ContainerStorage().FromContainerStorage(
		context.WithRole(api.RoleSliceElement),
		nil,
		element,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	var changed bool
	value, changed, err = toProviderValueSelected(
		context,
		children,
		owner,
		ownerBridge,
		profile,
		element,
		value,
	)
	if err != nil {
		return api.ExpressionEmission{}, false, err
	}
	value, err = providerContext.ContainerStorage().ToContainerStorage(
		providerContext.WithRole(api.RoleSliceElement),
		nil,
		element,
		value,
	)
	return value, changed, err
}

func sliceStorageZero(
	context api.Context,
	element types.Type,
) (api.ExpressionEmission, error) {
	zero, err := context.Values().Zero(context, nil, element)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.ContainerStorage().ToContainerStorage(
		context.WithRole(api.RoleSliceElement),
		nil,
		element,
		zero,
	)
}

func conversionArrow(
	context api.Context,
	parameter string,
	sourceType tsgo.TypeNode,
	targetType tsgo.TypeNode,
	value api.ExpressionEmission,
) tsgo.ArrowFunction {
	statements := append(value.Before(), context.Factory().ReturnStatement(value.Value()))
	return context.Factory().ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{context.Factory().ParameterDeclaration(
			nil,
			nil,
			context.Factory().Identifier(parameter),
			nil,
			sourceType,
			nil,
		)},
		targetType,
		context.Factory().EqualsGreaterThanToken(),
		context.Factory().Block(statements, true),
	)
}
