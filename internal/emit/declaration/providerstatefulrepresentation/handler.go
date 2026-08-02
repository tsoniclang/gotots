package providerstatefulrepresentation

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Build(
	factory tsgo.Factory,
	name string,
	typeTarget api.NameReference,
	valueTarget api.NameReference,
	typeArguments []tsgo.TypeNode,
	modifiers []tsgo.ModifierLike,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if name == "" || typeTarget.Name() == "" || valueTarget.Name() == "" {
		return nil, nil, &api.GeneratedArtifactShapeError{
			Artifact: name,
			Reason:   "provider stateful-representation target is invalid",
		}
	}
	statements := []tsgo.Statement{
		factory.TypeAliasDeclaration(
			modifiers,
			factory.Identifier(name),
			nil,
			factory.TypeReferenceNode(
				typeTarget.EntityName(factory),
				typeArguments,
			),
		),
		factory.VariableStatement(
			modifiers,
			factory.VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					factory.VariableDeclaration(
						factory.Identifier(name),
						nil,
						factory.TypeQueryNode(
							valueTarget.EntityName(factory),
							nil,
						),
						valueTarget.Expression(factory),
					),
				},
				tsgo.NodeFlagsConst,
			),
		),
	}
	return statements, api.CombineRequests(
		typeTarget.Requests(),
		valueTarget.Requests(),
	), nil
}
