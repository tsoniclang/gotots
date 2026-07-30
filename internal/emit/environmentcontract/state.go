package environmentcontract

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func StateField(
	context api.Context,
	children api.ChildEmitter,
	variable *types.Var,
) (tsgo.TypeElement, []api.RootRequest, error) {
	if variable == nil ||
		variable.IsField() ||
		variable.Pkg() == nil ||
		variable.Parent() != variable.Pkg().Scope() {
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "environment package variable is invalid",
		}
	}
	target, err := children.RepresentedType(
		context.WithRole(api.RolePackageVariableType),
		nil,
		variable.Type(),
	)
	if err != nil {
		return nil, nil, err
	}
	reference, err := context.Names().PackageVariable(variable)
	if err != nil {
		return nil, nil, err
	}
	return context.Factory().PropertySignatureDeclaration(
			nil,
			context.Factory().Identifier(reference.FieldName()),
			nil,
			target.Value(),
			context.Factory().OmittedExpression(),
		),
		target.Requests(),
		nil
}

func StateDeclaration(
	context api.Context,
	fields []tsgo.TypeElement,
) tsgo.Statement {
	return ambientConstant(
		context,
		"$state",
		context.Factory().TypeLiteralNode(fields),
	)
}
