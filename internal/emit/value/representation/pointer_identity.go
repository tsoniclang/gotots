package representation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	mapruntime "github.com/tsoniclang/gotots/internal/emit/runtime/map"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (owner Owner) directClassPointerHash(
	context api.Context,
	source ast.Node,
	element types.Type,
	value tsgo.Expression,
	storageIdentity bool,
	requests []api.RootRequest,
) (api.ExpressionEmission, error) {
	value, before, err := captureHashValue(context, value)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	identity := api.DirectExpression(value)
	usesStorageIdentity := false
	if storageIdentity {
		usesStorageIdentity, err = owner.directPointerUsesStorageIdentity(
			context,
			element,
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	if usesStorageIdentity {
		identity, err = owner.ToStorage(
			context,
			source,
			element,
			api.DirectExpression(value),
		)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		if len(identity.Before()) != 0 {
			return api.ExpressionEmission{}, &api.InvariantError{
				Role:   context.Role(),
				Reason: "direct pointer hash produced deferred storage work",
			}
		}
	}
	reference, err := context.Names().Runtime(
		api.RuntimeMapHash,
		api.ImportPhaseValue,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	undefined := context.Factory().Identifier("undefined")
	return api.NewExpressionEmission(
		before,
		context.Factory().ConditionalExpression(
			context.Factory().BinaryExpression(
				nil,
				value,
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				undefined,
			),
			context.Factory().QuestionToken(),
			context.Factory().NumericLiteral("0", tsgo.TokenFlagsNone),
			context.Factory().ColonToken(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					reference.Expression(context.Factory()),
					nil,
					context.Factory().Identifier(mapruntime.HashObjectMember),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{identity.Value()},
				tsgo.NodeFlagsNone,
			),
		),
		api.CombineRequests(
			identity.Requests(),
			reference.Requests(),
			requests,
		),
	)
}

func (owner Owner) directPointerUsesStorageIdentity(
	context api.Context,
	element types.Type,
) (bool, error) {
	typeName, _, ok := namedStruct(element)
	if !ok {
		return false, &api.InvariantError{
			Role:   context.Role(),
			Reason: "direct pointer element is not a named struct",
		}
	}
	providerOwned, err := context.Names().ProviderOwnedDeclaration(typeName)
	if err != nil {
		return false, err
	}
	if providerOwned {
		return false, nil
	}
	representation, err := context.Names().DefinedValueRepresentation(typeName)
	if err != nil {
		return false, err
	}
	return representation.Kind() ==
		api.DefinedValueRepresentationGeneratedWrapper, nil
}

func captureHashValue(
	context api.Context,
	value tsgo.Expression,
) (tsgo.Expression, []tsgo.Statement, error) {
	if value.Kind() == tsgo.SyntaxKindIdentifier {
		return value, nil, nil
	}
	name, err := context.Names().Temporary(api.TemporaryEqualityOperand)
	if err != nil {
		return nil, nil, err
	}
	statement := context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{context.Factory().VariableDeclaration(
				context.Factory().Identifier(name),
				nil,
				nil,
				value,
			)},
			tsgo.NodeFlagsConst,
		),
	)
	return context.Factory().Identifier(name), []tsgo.Statement{statement}, nil
}
