package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitValueOperation(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	className string,
	classType tsgo.TypeNode,
	fields []field,
	operation api.NamedStructOperation,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	memberName, err := api.NamedStructOperationMemberName(operation)
	if err != nil {
		return nil, nil, err
	}
	switch operation {
	case api.NamedStructOperationZero:
		return zeroMethod(context, source, memberName, className, classType, fields)
	case api.NamedStructOperationCopy:
		return copyMethod(context, source, memberName, className, classType, fields)
	case api.NamedStructOperationEqual:
		return equalMethod(
			context,
			children,
			source,
			memberName,
			classType,
			fields,
		)
	case api.NamedStructOperationHash:
		return hashMethod(context, source, memberName, classType, fields)
	case api.NamedStructOperationConvert:
		return conversionMethod(
			context,
			children,
			source,
			memberName,
			className,
			classType,
			fields,
		)
	default:
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "named-struct operation is invalid",
		}
	}
}

func hashMethod(
	context api.Context,
	source ast.Node,
	memberName string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	runtime, err := context.Names().Runtime(
		api.RuntimeMapHash,
		api.ImportPhaseValue,
	)
	if err != nil {
		return nil, nil, err
	}
	hashName := "$hash"
	hash := context.Factory().Identifier(hashName)
	body := []tsgo.Statement{
		context.Factory().VariableStatement(
			nil,
			context.Factory().VariableDeclarationList(
				[]tsgo.VariableDeclaration{
					context.Factory().VariableDeclaration(
						hash,
						nil,
						nil,
						context.Factory().NumericLiteral(
							"2166136261",
							tsgo.TokenFlagsNone,
						),
					),
				},
				tsgo.NodeFlagsLet,
			),
		),
	}
	requests := runtime.Requests()
	for _, field := range fields {
		if field.blank {
			continue
		}
		fieldHash, err := context.Values().Hash(
			context.WithRole(api.RoleStructHashField),
			field.source,
			field.object.Type(),
			property(context, "$source", field.name),
		)
		if err != nil {
			return nil, nil, err
		}
		body = append(body, fieldHash.Before()...)
		body = append(
			body,
			context.Factory().ExpressionStatement(
				context.Factory().BinaryExpression(
					nil,
					hash,
					nil,
					context.Factory().BinaryOperatorToken(
						tsgo.BinaryOperatorEqualsToken,
					),
					context.Factory().CallExpression(
						context.Factory().PropertyAccessExpression(
							context.Factory().Identifier(runtime.Name()),
							nil,
							context.Factory().Identifier(
								mapruntime.HashMixMember,
							),
							tsgo.NodeFlagsNone,
						),
						nil,
						nil,
						[]tsgo.Expression{hash, fieldHash.Value()},
						tsgo.NodeFlagsNone,
					),
				),
			),
		)
		requests = append(requests, fieldHash.Requests()...)
	}
	body = append(body, context.Factory().ReturnStatement(hash))
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		},
		context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		),
		body,
	), requests, nil
}

func zeroMethod(
	context api.Context,
	source ast.Node,
	memberName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(fields))
	var requests []api.RootRequest
	for _, field := range fields {
		value, err := context.Values().Zero(
			context.WithRole(api.RoleStructZeroField),
			field.source,
			field.object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		if len(value.Before()) != 0 {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleStructZeroField),
				api.CategoryDeclaration,
				source,
			)
		}
		arguments = append(arguments, value.Value())
		requests = append(requests, value.Requests()...)
	}
	return operationMethod(
		context,
		memberName,
		nil,
		classType,
		[]tsgo.Statement{context.Factory().ReturnStatement(
			context.Factory().NewExpression(
				context.Factory().Identifier(className),
				nil,
				arguments,
			),
		)},
	), requests, nil
}

func copyMethod(
	context api.Context,
	source ast.Node,
	memberName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	arguments := make([]tsgo.Expression, 0, len(fields))
	var requests []api.RootRequest
	for _, field := range fields {
		var copied api.ExpressionEmission
		var err error
		if field.blank {
			copied, err = context.Values().Zero(
				context.WithRole(api.RoleStructCopyField),
				field.source,
				field.object.Type(),
			)
		} else {
			value := api.DirectExpression(property(context, "$source", field.name))
			copied, err = context.Values().Copy(
				context.WithRole(api.RoleStructCopyField),
				field.source,
				field.object.Type(),
				value,
			)
		}
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
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{parameter(context, "$source", classType)},
		classType,
		[]tsgo.Statement{context.Factory().ReturnStatement(
			context.Factory().NewExpression(
				context.Factory().Identifier(className),
				nil,
				arguments,
			),
		)},
	), requests, nil
}

func equalMethod(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	memberName string,
	classType tsgo.TypeNode,
	fields []field,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	equalities := make([]api.ExpressionEmission, 0, len(fields))
	var requests []api.RootRequest
	hasPrerequisites := false
	for _, field := range fields {
		if field.blank {
			continue
		}
		equal, err := context.Values().Equal(
			context.WithRole(api.RoleStructEqualField),
			field.source,
			field.object.Type(),
			property(context, "$left", field.name),
			property(context, "$right", field.name),
		)
		if err != nil {
			return nil, nil, err
		}
		if len(equal.Before()) != 0 {
			hasPrerequisites = true
		}
		equalities = append(equalities, equal)
		requests = append(requests, equal.Requests()...)
	}
	resultType, err := children.RepresentedType(
		context.WithRole(api.RoleResultType),
		source,
		types.Typ[types.Bool],
	)
	if err != nil {
		return nil, nil, err
	}
	requests = append(requests, resultType.Requests()...)
	body := structEqualityBody(context, equalities, hasPrerequisites)
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{
			parameter(context, "$left", classType),
			parameter(context, "$right", classType),
		},
		resultType.Value(),
		body,
	), requests, nil
}

func structEqualityBody(
	context api.Context,
	equalities []api.ExpressionEmission,
	hasPrerequisites bool,
) []tsgo.Statement {
	if !hasPrerequisites {
		var expression tsgo.Expression = context.Factory().TrueLiteral()
		for index, equal := range equalities {
			if index == 0 {
				expression = equal.Value()
				continue
			}
			expression = context.Factory().BinaryExpression(
				nil,
				expression,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorAmpersandAmpersandToken,
				),
				equal.Value(),
			)
		}
		return []tsgo.Statement{
			context.Factory().ReturnStatement(expression),
		}
	}
	var body []tsgo.Statement
	for _, equal := range equalities {
		body = append(body, equal.Before()...)
		body = append(
			body,
			context.Factory().IfStatement(
				context.Factory().PrefixUnaryExpression(
					tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
					equal.Value(),
				),
				context.Factory().Block(
					[]tsgo.Statement{
						context.Factory().ReturnStatement(
							context.Factory().FalseLiteral(),
						),
					},
					true,
				),
				nil,
			),
		)
	}
	return append(
		body,
		context.Factory().ReturnStatement(context.Factory().TrueLiteral()),
	)
}

func operationMethod(
	context api.Context,
	name string,
	parameters []tsgo.ParameterDeclaration,
	result tsgo.TypeNode,
	statements []tsgo.Statement,
) tsgo.MethodDeclaration {
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{context.Factory().StaticKeyword()},
		nil,
		context.Factory().Identifier(name),
		nil,
		nil,
		parameters,
		result,
		context.Factory().Block(statements, true),
	)
}

func parameter(
	context api.Context,
	name string,
	targetType tsgo.TypeNode,
) tsgo.ParameterDeclaration {
	return context.Factory().ParameterDeclaration(
		nil,
		nil,
		context.Factory().Identifier(name),
		nil,
		targetType,
		nil,
	)
}

func property(
	context api.Context,
	receiver string,
	name string,
) tsgo.PropertyAccessExpression {
	return context.Factory().PropertyAccessExpression(
		context.Factory().Identifier(receiver),
		nil,
		context.Factory().Identifier(name),
		tsgo.NodeFlagsNone,
	)
}
