package declaration

import (
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type TypeParameterLayout struct {
	parameters []tsgo.TypeParameterDeclaration
	arguments  []tsgo.TypeNode
}

func EmitTypeParameterLayout(
	context api.Context,
	profile api.GenericRepresentationProfile,
	logicalNames map[*types.TypeParam]string,
) (TypeParameterLayout, error) {
	parameters := profile.Parameters()
	if len(parameters) == 0 || len(logicalNames) != len(parameters) {
		return TypeParameterLayout{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "generic target parameter layout is incomplete",
		}
	}
	layout := TypeParameterLayout{
		parameters: make(
			[]tsgo.TypeParameterDeclaration,
			0,
			len(parameters),
		),
		arguments: make([]tsgo.TypeNode, 0, len(parameters)),
	}
	for _, parameter := range parameters {
		name := logicalNames[parameter]
		if name == "" {
			return TypeParameterLayout{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic target parameter name is missing",
			}
		}
		layout.append(context, name)
	}
	return layout, nil
}

func (l *TypeParameterLayout) append(context api.Context, name string) {
	l.parameters = append(
		l.parameters,
		context.Factory().TypeParameterDeclaration(
			nil,
			context.Factory().Identifier(name),
			nil,
			nil,
			nil,
		),
	)
	l.arguments = append(l.arguments, context.Factory().TypeReferenceNode(
		context.Factory().Identifier(name),
		nil,
	))
}

func (l TypeParameterLayout) Parameters() []tsgo.TypeParameterDeclaration {
	return slices.Clone(l.parameters)
}

func (l TypeParameterLayout) Arguments() []tsgo.TypeNode {
	return slices.Clone(l.arguments)
}
