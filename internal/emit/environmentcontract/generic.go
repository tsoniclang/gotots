package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
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
	requirements []api.DeclarationRequirement,
) (genericScope, error) {
	parameters := api.GenericDeclarationParameters(owner)
	if len(parameters) == 0 {
		return genericScope{context: context}, nil
	}
	names := make(map[*types.TypeParam]string, len(parameters))
	profile, err := api.SelectGenericRepresentationProfile(
		owner,
		requirements,
	)
	if err != nil {
		return genericScope{}, err
	}
	for index, parameter := range parameters {
		name := genericName(index)
		names[parameter] = name
	}
	layout, err := genericdeclaration.EmitTypeParameterLayout(
		context,
		profile,
		names,
	)
	if err != nil {
		return genericScope{}, err
	}
	context, err = context.WithEnvironmentGenericParameters(owner, names)
	if err != nil {
		return genericScope{}, err
	}
	return genericScope{
		context:    context,
		parameters: layout.Parameters(),
		arguments:  layout.Arguments(),
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
