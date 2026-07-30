package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	genericabi "github.com/tsoniclang/gotots/internal/emit/generic/abi"
	genericdeclaration "github.com/tsoniclang/gotots/internal/emit/generic/declaration"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitValueOperation(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	className string,
	classType tsgo.TypeNode,
	fields []field,
	assembly operationAssembly,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
	canonicalStorage bool,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	operation := assembly.operation
	memberName, err := api.NamedStructOperationMemberName(operation)
	if err != nil {
		return nil, nil, err
	}
	context, capabilities, capabilityRequests, err :=
		prepareOperation(
			context,
			children,
			source,
			assembly,
			typeParameters,
		)
	if err != nil {
		return nil, nil, err
	}
	var member tsgo.MethodDeclaration
	var requests []api.RootRequest
	switch operation {
	case api.NamedStructOperationZero:
		member, requests, err = zeroMethod(
			context,
			source,
			memberName,
			className,
			classType,
			fields,
			capabilities,
			typeParameters,
			typeArguments,
			canonicalStorage,
		)
	case api.NamedStructOperationCopy:
		member, requests, err = copyMethod(
			context,
			source,
			memberName,
			className,
			classType,
			fields,
			capabilities,
			typeParameters,
			typeArguments,
			canonicalStorage,
		)
	case api.NamedStructOperationEqual:
		member, requests, err = equalMethod(
			context,
			children,
			source,
			memberName,
			classType,
			fields,
			capabilities,
			typeParameters,
			canonicalStorage,
		)
	case api.NamedStructOperationHash:
		member, requests, err = hashMethod(
			context,
			source,
			memberName,
			classType,
			fields,
			capabilities,
			typeParameters,
			canonicalStorage,
		)
	case api.NamedStructOperationConvert:
		member, requests, err = conversionMethod(
			context,
			children,
			source,
			memberName,
			className,
			classType,
			fields,
			capabilities,
			typeParameters,
			typeArguments,
			canonicalStorage,
		)
	default:
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "named-struct operation is invalid",
		}
	}
	if err != nil {
		return nil, nil, err
	}
	return member, api.CombineRequests(capabilityRequests, requests), nil
}

func prepareOperation(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	assembly operationAssembly,
	typeParameters []tsgo.TypeParameterDeclaration,
) (
	api.Context,
	[]tsgo.ParameterDeclaration,
	[]api.RootRequest,
	error,
) {
	if len(typeParameters) == 0 {
		if len(assembly.capabilities) != 0 {
			return api.Context{}, nil, nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "non-generic struct operation received generic capabilities",
			}
		}
		return context, nil, nil, nil
	}
	context = context.WithGenericNamedStructOperation(assembly.operation)
	bindings, requests, err := genericdeclaration.EmitOperationParameters(
		context,
		children,
		source,
		assembly.capabilities,
	)
	if err != nil {
		return api.Context{}, nil, nil, err
	}
	if len(assembly.capabilities) == 0 {
		return context, nil, requests, nil
	}
	capabilities, err := genericabi.JoinCapabilities(
		assembly.capabilities[0].Owner(),
		assembly.capabilities,
		bindings,
	)
	if err != nil {
		return api.Context{}, nil, nil, err
	}
	return context, capabilities, requests, nil
}

func zeroMethod(
	context api.Context,
	source ast.Node,
	memberName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
	canonicalStorage bool,
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
		value, err = operationConstructionValue(
			context.WithRole(api.RoleStructZeroField),
			field.source,
			field,
			value,
			canonicalStorage,
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
			construct(context, className, typeArguments, arguments),
		)},
		capabilities,
		typeParameters,
	), requests, nil
}

func copyMethod(
	context api.Context,
	source ast.Node,
	memberName string,
	className string,
	classType tsgo.TypeNode,
	fields []field,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
	canonicalStorage bool,
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
			value, valueErr := operationFieldValue(
				context.WithRole(api.RoleStructCopyField),
				field.source,
				"$source",
				field,
				canonicalStorage,
			)
			if valueErr != nil {
				return nil, nil, valueErr
			}
			copied, err = context.Values().Transfer(
				context.WithRole(api.RoleStructCopyField),
				field.source,
				field.object.Type(),
				field.object.Type(),
				api.ValueTransferCopy,
				value,
			)
			if err == nil {
				copied, err = api.NewExpressionEmission(
					copied.Before(),
					copied.Value(),
					api.CombineRequests(
						value.Requests(),
						copied.Requests(),
					),
				)
			}
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
	return operationMethod(
		context,
		memberName,
		[]tsgo.ParameterDeclaration{parameter(context, "$source", classType)},
		classType,
		[]tsgo.Statement{context.Factory().ReturnStatement(
			construct(context, className, typeArguments, arguments),
		)},
		capabilities,
		typeParameters,
	), requests, nil
}

func equalMethod(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	memberName string,
	classType tsgo.TypeNode,
	fields []field,
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
	canonicalStorage bool,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	equalities := make([]api.ExpressionEmission, 0, len(fields))
	var requests []api.RootRequest
	hasPrerequisites := false
	for _, field := range fields {
		if field.blank {
			continue
		}
		left, err := operationFieldValue(
			context.WithRole(api.RoleStructEqualField),
			field.source,
			"$left",
			field,
			canonicalStorage,
		)
		if err != nil {
			return nil, nil, err
		}
		right, err := operationFieldValue(
			context.WithRole(api.RoleStructEqualField),
			field.source,
			"$right",
			field,
			canonicalStorage,
		)
		if err != nil {
			return nil, nil, err
		}
		equal, err := context.Values().Equal(
			context.WithRole(api.RoleStructEqualField),
			field.source,
			field.object.Type(),
			left.Value(),
			right.Value(),
		)
		if err != nil {
			return nil, nil, err
		}
		if len(equal.Before()) != 0 {
			hasPrerequisites = true
		}
		equal, err = api.NewExpressionEmission(
			append(
				append(left.Before(), right.Before()...),
				equal.Before()...,
			),
			equal.Value(),
			api.CombineRequests(
				left.Requests(),
				right.Requests(),
				equal.Requests(),
			),
		)
		if err != nil {
			return nil, nil, err
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
		capabilities,
		typeParameters,
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
	capabilities []tsgo.ParameterDeclaration,
	typeParameters []tsgo.TypeParameterDeclaration,
) tsgo.MethodDeclaration {
	parameters = append(
		append([]tsgo.ParameterDeclaration(nil), capabilities...),
		parameters...,
	)
	return context.Factory().MethodDeclaration(
		[]tsgo.ModifierLike{context.Factory().StaticKeyword()},
		nil,
		context.Factory().Identifier(name),
		nil,
		typeParameters,
		parameters,
		result,
		context.Factory().Block(statements, true),
	)
}

func construct(
	context api.Context,
	className string,
	typeArguments []tsgo.TypeNode,
	arguments []tsgo.Expression,
) tsgo.CallExpression {
	return context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(className),
			nil,
			context.Factory().Identifier(api.StructMakeMember),
			tsgo.NodeFlagsNone,
		),
		nil,
		typeArguments,
		arguments,
		tsgo.NodeFlagsNone,
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
