package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type genericScope struct {
	context    api.Context
	parameters []tsgo.TypeParameterDeclaration
	arguments  []tsgo.TypeNode
}

func enterGeneric(
	context api.Context,
	owner types.Object,
) (genericScope, error) {
	parameters := api.GenericDeclarationParameters(owner)
	if len(parameters) == 0 {
		return genericScope{context: context}, nil
	}
	names := make(map[*types.TypeParam]string, len(parameters))
	targetParameters := make(
		[]tsgo.TypeParameterDeclaration,
		0,
		len(parameters),
	)
	arguments := make([]tsgo.TypeNode, 0, len(parameters))
	for index, parameter := range parameters {
		name := genericName(index)
		names[parameter] = name
		targetParameters = append(
			targetParameters,
			context.Factory().TypeParameterDeclaration(
				nil,
				context.Factory().Identifier(name),
				nil,
				nil,
				nil,
			),
		)
		arguments = append(arguments, context.Factory().TypeReferenceNode(
			context.Factory().Identifier(name),
			nil,
		))
	}
	context, err := context.WithEnvironmentGenericParameters(owner, names)
	if err != nil {
		return genericScope{}, err
	}
	return genericScope{
		context:    context,
		parameters: targetParameters,
		arguments:  arguments,
	}, nil
}

func genericName(index int) string {
	const digits = "0123456789"
	if index < 10 {
		return "$T" + string(digits[index])
	}
	result := ""
	for index != 0 {
		result = string(digits[index%10]) + result
		index /= 10
	}
	return "$T" + result
}
