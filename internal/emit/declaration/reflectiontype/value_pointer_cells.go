package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func pointerCellValueProperties(
	context api.Context,
	names api.ReflectionNames,
	reflectionType *types.TypeName,
	pointee types.Type,
	scaffold *locationScaffold,
) ([]tsgo.ObjectLiteralElementLike, error) {
	factory := scaffold.factory
	sliceAdapter, err := context.Names().InterfaceAdapter(pointee, nil)
	if err != nil {
		return nil, err
	}
	descriptor, err := names.ReflectionValueType(pointee, reflectionType)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, sliceAdapter.Requests()...)
	scaffold.requests = append(scaffold.requests, descriptor.Requests()...)
	cellValue := memberAccess(factory, "instance", "value")
	cellWrite := guardedForeignPayload(
		scaffold,
		sliceAdapter,
		"Value.Set",
	)
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			sliceAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{cellValue},
		),
		set: factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.BinaryExpression(
				nil,
				cellValue,
				nil,
				factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
				cellWrite,
			)),
		}, true),
	})
	body := factory.Block([]tsgo.Statement{
		foreignBoxGuardStatement(scaffold, "Value.Elem"),
		constStatement(factory, "instance", boxPayload(factory)),
		factory.IfStatement(
			factory.BinaryExpression(
				nil,
				factory.Identifier("instance"),
				nil,
				factory.BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsEqualsEqualsToken,
				),
				factory.Identifier("undefined"),
			),
			factory.Block([]tsgo.Statement{
				factory.ReturnStatement(factory.Identifier("undefined")),
			}, true),
			nil,
		),
		factory.ReturnStatement(location),
	}, true)
	elem := factory.ArrowFunction(
		nil,
		nil,
		[]tsgo.ParameterDeclaration{boxParameter(scaffold)},
		nil,
		factory.EqualsGreaterThanToken(),
		body,
	)
	properties := []tsgo.ObjectLiteralElementLike{
		expressionProperty(factory, "elem", elem),
		pointerZeroProperty(scaffold),
	}
	if pointeeBasic, basicOK := types.Unalias(pointee).
		Underlying().(*types.Basic); basicOK {
		pointeeZero, zeroErr := scalarZeroExpression(
			context,
			factory,
			pointeeBasic,
		)
		if zeroErr != nil {
			return nil, zeroErr
		}
		if pointeeZero != nil {
			logicalZero, zeroRequests, constructErr := constructedScalarValue(
				context,
				pointee,
				pointeeZero,
			)
			if constructErr != nil {
				return nil, constructErr
			}
			runtimePointer, pointerErr := context.Names().Runtime(
				api.RuntimePointer,
				api.ImportPhaseValue,
			)
			if pointerErr != nil {
				return nil, pointerErr
			}
			scaffold.requests = append(
				scaffold.requests,
				runtimePointer.Requests()...,
			)
			scaffold.requests = append(
				scaffold.requests,
				zeroRequests...,
			)
			properties = append(properties, expressionProperty(
				factory,
				"newPointer",
				factory.ArrowFunction(
					nil,
					nil,
					nil,
					factory.TypeReferenceNode(
						scaffold.boxType.EntityName(factory),
						nil,
					),
					factory.EqualsGreaterThanToken(),
					factory.ParenthesizedExpression(
						factory.NewExpression(
							scaffold.adapter.Expression(factory),
							nil,
							[]tsgo.Expression{factory.CallExpression(
								factory.PropertyAccessExpression(
									runtimePointer.Expression(factory),
									nil,
									factory.Identifier("cell"),
									tsgo.NodeFlagsNone,
								),
								nil,
								nil,
								[]tsgo.Expression{logicalZero},
								tsgo.NodeFlagsNone,
							)},
						),
					),
				),
			))
		}
	}
	return properties, nil
}
