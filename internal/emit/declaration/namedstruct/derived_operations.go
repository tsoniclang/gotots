package namedstruct

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitDerivedValueOperation(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	basis types.Type,
	className string,
	classType tsgo.TypeNode,
	storageType tsgo.TypeNode,
	assembly operationAssembly,
	typeParameters []tsgo.TypeParameterDeclaration,
	typeArguments []tsgo.TypeNode,
) (tsgo.MethodDeclaration, []api.RootRequest, error) {
	memberName, err := api.NamedStructOperationMemberName(assembly.operation)
	if err != nil {
		return nil, nil, err
	}
	context, capabilities, capabilityRequests, err := prepareOperation(
		context,
		children,
		source,
		assembly,
		typeParameters,
	)
	if err != nil {
		return nil, nil, err
	}
	var parameters []tsgo.ParameterDeclaration
	var result tsgo.TypeNode
	var body []tsgo.Statement
	var requests []api.RootRequest
	switch assembly.operation {
	case api.NamedStructOperationZero:
		result = classType
		body, requests, err = derivedConstructBody(
			context,
			source,
			basis,
			className,
			storageType,
			nil,
			nil,
			typeArguments,
		)
	case api.NamedStructOperationCopy:
		parameters = []tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		}
		result = classType
		body, requests, err = derivedCopyBody(
			context,
			source,
			basis,
			className,
			typeArguments,
		)
	case api.NamedStructOperationEqual:
		parameters = []tsgo.ParameterDeclaration{
			parameter(context, "$left", classType),
			parameter(context, "$right", classType),
		}
		result = context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindBooleanKeyword,
		)
		body, requests, err = derivedEqualBody(
			context,
			source,
			basis,
		)
	case api.NamedStructOperationHash:
		parameters = []tsgo.ParameterDeclaration{
			parameter(context, "$source", classType),
		}
		result = context.Factory().KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindNumberKeyword,
		)
		body, requests, err = derivedHashBody(
			context,
			source,
			basis,
		)
	case api.NamedStructOperationConvert:
		basisType, typeErr := children.RepresentedType(
			context.WithRole(api.RoleDefinedUnderlyingType),
			source,
			basis,
		)
		if typeErr != nil {
			return nil, nil, typeErr
		}
		parameters = []tsgo.ParameterDeclaration{
			parameter(context, "$source", basisType.Value()),
		}
		result = classType
		body, requests, err = derivedConvertBody(
			context,
			source,
			basis,
			className,
			typeArguments,
		)
		requests = api.CombineRequests(requests, basisType.Requests())
	default:
		return nil, nil, &api.InvariantError{
			Role:   context.Role(),
			Reason: "derived-struct operation is invalid",
		}
	}
	if err != nil {
		return nil, nil, err
	}
	return operationMethod(
		context,
		memberName,
		parameters,
		result,
		body,
		capabilities,
		typeParameters,
	), api.CombineRequests(capabilityRequests, requests), nil
}

func derivedCopyBody(
	context api.Context,
	source ast.Node,
	basis types.Type,
	className string,
	typeArguments []tsgo.TypeNode,
) ([]tsgo.Statement, []api.RootRequest, error) {
	restored, err := restoreDerivedParameter(
		context,
		source,
		basis,
		"$source",
	)
	if err != nil {
		return nil, nil, err
	}
	body, basisValue := captureDerivedValue(context, "$basis", restored)
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleStructCopyField),
		source,
		basis,
		basis,
		api.ValueTransferCopy,
		api.DirectExpression(basisValue),
	)
	if err != nil {
		return nil, nil, err
	}
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		basis,
		copied,
	)
	if err != nil {
		return nil, nil, err
	}
	body = append(body, stored.Before()...)
	body = append(body, context.Factory().ReturnStatement(
		context.Factory().NewExpression(
			context.Factory().Identifier(className),
			typeArguments,
			[]tsgo.Expression{stored.Value()},
		),
	))
	return body, api.CombineRequests(
		restored.Requests(),
		copied.Requests(),
		stored.Requests(),
	), nil
}

func derivedConvertBody(
	context api.Context,
	source ast.Node,
	basis types.Type,
	className string,
	typeArguments []tsgo.TypeNode,
) ([]tsgo.Statement, []api.RootRequest, error) {
	copied, err := context.Values().Transfer(
		context.WithRole(api.RoleConversionOperand),
		source,
		basis,
		basis,
		api.ValueTransferCopy,
		api.DirectExpression(context.Factory().Identifier("$source")),
	)
	if err != nil {
		return nil, nil, err
	}
	stored, err := context.Values().ToStorage(
		context.WithRole(api.RoleStorageType),
		source,
		basis,
		copied,
	)
	if err != nil {
		return nil, nil, err
	}
	body := append([]tsgo.Statement(nil), stored.Before()...)
	body = append(body, context.Factory().ReturnStatement(
		context.Factory().NewExpression(
			context.Factory().Identifier(className),
			typeArguments,
			[]tsgo.Expression{stored.Value()},
		),
	))
	return body, api.CombineRequests(copied.Requests(), stored.Requests()), nil
}

func derivedEqualBody(
	context api.Context,
	source ast.Node,
	basis types.Type,
) ([]tsgo.Statement, []api.RootRequest, error) {
	left, err := restoreDerivedParameter(context, source, basis, "$left")
	if err != nil {
		return nil, nil, err
	}
	right, err := restoreDerivedParameter(context, source, basis, "$right")
	if err != nil {
		return nil, nil, err
	}
	body, leftValue := captureDerivedValue(context, "$leftBasis", left)
	rightBody, rightValue := captureDerivedValue(context, "$rightBasis", right)
	body = append(body, rightBody...)
	equal, err := context.Values().Equal(
		context.WithRole(api.RoleStructEqualField),
		source,
		basis,
		leftValue,
		rightValue,
	)
	if err != nil {
		return nil, nil, err
	}
	body = append(body, equal.Before()...)
	body = append(body, context.Factory().ReturnStatement(equal.Value()))
	return body, api.CombineRequests(
		left.Requests(),
		right.Requests(),
		equal.Requests(),
	), nil
}

func derivedHashBody(
	context api.Context,
	source ast.Node,
	basis types.Type,
) ([]tsgo.Statement, []api.RootRequest, error) {
	restored, err := restoreDerivedParameter(
		context,
		source,
		basis,
		"$source",
	)
	if err != nil {
		return nil, nil, err
	}
	body, basisValue := captureDerivedValue(context, "$basis", restored)
	hash, err := context.Values().Hash(
		context.WithRole(api.RoleStructHashField),
		source,
		basis,
		basisValue,
	)
	if err != nil {
		return nil, nil, err
	}
	body = append(body, hash.Before()...)
	body = append(body, context.Factory().ReturnStatement(hash.Value()))
	return body, api.CombineRequests(
		restored.Requests(),
		hash.Requests(),
	), nil
}

func restoreDerivedParameter(
	context api.Context,
	source ast.Node,
	basis types.Type,
	name string,
) (api.ExpressionEmission, error) {
	return context.Values().FromStorage(
		context.WithRole(api.RoleStorageType),
		source,
		basis,
		api.DirectExpression(context.Factory().PropertyAccessExpression(
			context.Factory().Identifier(name),
			nil,
			context.Factory().Identifier(derivedStorageMember),
			tsgo.NodeFlagsNone,
		)),
	)
}

func captureDerivedValue(
	context api.Context,
	name string,
	value api.ExpressionEmission,
) ([]tsgo.Statement, tsgo.Identifier) {
	target := context.Factory().Identifier(name)
	body := append([]tsgo.Statement(nil), value.Before()...)
	body = append(body, context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				target,
				nil,
				nil,
				value.Value(),
			)},
			tsgo.NodeFlagsConst,
		),
	))
	return body, target
}
