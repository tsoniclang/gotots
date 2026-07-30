package defined

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (api.TypeEmission, bool, error) {
	model, ok := Resolve(sourceType)
	if !ok {
		return api.TypeEmission{}, false, nil
	}
	reference, err := context.Names().TypeReference(model.TypeName())
	if err != nil {
		return api.TypeEmission{}, true, err
	}
	arguments := make(
		[]tsgo.TypeNode,
		0,
		model.Type().TypeArgs().Len(),
	)
	var requests []api.RootRequest
	for index := range model.Type().TypeArgs().Len() {
		argument, err := children.RepresentedType(
			context.WithRole(api.RoleDefinedTypeArgument),
			source,
			model.Type().TypeArgs().At(index),
		)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
		arguments = append(arguments, argument.Value())
		requests = append(requests, argument.Requests()...)
	}
	target := tsgo.TypeNode(context.Factory().TypeReferenceNode(
		context.Factory().Identifier(reference.Name()),
		arguments,
	))
	if _, profiled := context.GenericCallableProfile(); profiled {
		underlying, err := profiledUnderlyingType(
			context,
			children,
			source,
			model,
		)
		if err != nil {
			return api.TypeEmission{}, true, err
		}
		target = context.Factory().IntersectionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().TypeLiteralNode([]tsgo.TypeElement{
				context.Factory().PropertySignatureDeclaration(
					[]tsgo.ModifierLike{
						context.Factory().ReadonlyKeyword(),
					},
					context.Factory().Identifier(ValueMember),
					nil,
					underlying.Value(),
					context.Factory().OmittedExpression(),
				),
			}),
		})
		requests = append(requests, underlying.Requests()...)
	}
	if model.NilCapable() {
		target = context.Factory().UnionTypeNode([]tsgo.TypeNode{
			target,
			context.Factory().KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindUndefinedKeyword,
			),
		})
	}
	return api.DirectType(
		target,
		api.CombineRequests(reference.Requests(), requests)...,
	), true, nil
}

func profiledUnderlyingType(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	model Model,
) (api.TypeEmission, error) {
	if signature, callableFamily := model.Callable(); callableFamily {
		return callable.EmitDefinedNonNilType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			children,
			source,
			signature,
		)
	}
	return children.RepresentedType(
		context.WithRole(api.RoleDefinedUnderlyingType),
		source,
		model.Underlying(),
	)
}
