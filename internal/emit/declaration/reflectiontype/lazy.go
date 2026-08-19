package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func registrationExpression(
	context api.Context,
	names api.ReflectionNames,
	sourceType types.Type,
	reflectionType *types.TypeName,
	descriptorType api.NameReference,
) (tsgo.ObjectLiteralExpression, []api.RootRequest, error) {
	factory := context.Factory()
	var properties []tsgo.ObjectLiteralElementLike
	var requests []api.RootRequest
	if interfaceDynamicType(sourceType) {
		dynamicType, err := context.Names().InterfaceDynamicType(sourceType)
		if err != nil {
			return nil, nil, err
		}
		properties = append(properties, expressionProperty(
			factory,
			"dynamicType",
			dynamicType.Expression(factory),
		))
		requests = append(requests, dynamicType.Requests()...)
	}
	if pointer, ok := types.Unalias(sourceType).Underlying().(*types.Pointer); ok {
		element, err := names.ReflectionType(pointer.Elem(), reflectionType)
		if err != nil {
			return nil, nil, err
		}
		properties = append(properties, expressionProperty(
			factory,
			"pointerElement",
			arrow(factory, descriptorType, element.Expression(factory)),
		))
		requests = append(requests, element.Requests()...)
	}
	if len(properties) == 0 {
		return nil, requests, nil
	}
	return factory.ObjectLiteralExpression(properties, true), requests, nil
}

func deferred(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.ArrowFunction {
	return factory.ArrowFunction(
		nil,
		nil,
		nil,
		nil,
		factory.EqualsGreaterThanToken(),
		factory.ParenthesizedExpression(value),
	)
}
