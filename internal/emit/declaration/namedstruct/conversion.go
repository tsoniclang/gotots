package namedstruct

import (
	"go/ast"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func conversionMethod(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	memberName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
	constructionTypes []tsgo.TypeNode,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
	canonicalStorage bool,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	sourceMembers := make([]tsgo.TypeElement, 0, len(fields))
	arguments := make([]tsgo.Expression, 0, len(fields))
	var requests []api.RootRequest
	for _, field := range fields {
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleStructFieldType),
			field.typeSource,
			field.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		requests = append(requests, targetType.Requests()...)
		if !field.blank {
			sourceMembers = append(
				sourceMembers,
				context.Factory().PropertySignatureDeclaration(
					nil,
					context.Factory().Identifier(field.name),
					nil,
					targetType.Value(),
					context.Factory().OmittedExpression(),
				),
			)
		}

		var copied api.ExpressionEmission
		if field.blank {
			copied, err = context.Values().Zero(
				context.WithRole(api.RoleStructCopyField),
				field.source,
				field.object.Type(),
			)
		} else {
			copied, err = context.Values().Transfer(
				context.WithRole(api.RoleStructCopyField),
				field.source,
				field.object.Type(),
				field.object.Type(),
				api.ValueTransferCopy,
				api.DirectExpression(property(
					context,
					"$source",
					field.name,
				)),
			)
		}
		if err != nil {
			return nil, nil, err
		}
		copied, err = operationConstructionValue(
			context.WithRole(api.RoleStructCopyField),
			field.source,
			field,
			copied,
			canonicalStorage,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(copied.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleStructCopyField),
				api.CategoryDeclaration,
				source,
			)
		}
		arguments = append(arguments, copied.Value())
		requests = append(requests, copied.Requests()...)
	}
	sourceType := context.Factory().TypeLiteralNode(sourceMembers)
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{
			parameter(context, "$source", sourceType),
		},
		classType,
		[]tsgo.Statement{context.Factory().ReturnStatement(
			construct(
				context,
				className,
				typeArguments,
				fields,
				constructionTypes,
				arguments,
			),
		)},
		capabilities,
		typeParameters,
	), requests, nil
}
