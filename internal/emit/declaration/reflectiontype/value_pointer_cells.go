package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
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
	loadPointer, err := context.Names().TsonicCore(tsoniccore.SymbolLoadPointer)
	if err != nil {
		return nil, err
	}
	storePointer, err := context.Names().TsonicCore(tsoniccore.SymbolStorePointer)
	if err != nil {
		return nil, err
	}
	scaffold.requests = append(scaffold.requests, loadPointer.Requests()...)
	scaffold.requests = append(scaffold.requests, storePointer.Requests()...)
	cellRead := factory.CallExpression(
		loadPointer.Expression(factory),
		nil,
		nil,
		[]tsgo.Expression{factory.Identifier("instance")},
		tsgo.NodeFlagsNone,
	)
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			sliceAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{cellRead},
		),
		set: factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.CallExpression(
				storePointer.Expression(factory),
				nil,
				nil,
				[]tsgo.Expression{
					factory.Identifier("instance"),
					guardedForeignPayload(
						scaffold,
						sliceAdapter,
						"Value.Set",
					),
				},
				tsgo.NodeFlagsNone,
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
		supportedZero, zeroErr := scalarZeroExpression(
			context,
			factory,
			pointeeBasic,
		)
		if zeroErr != nil {
			return nil, zeroErr
		}
		if supportedZero == nil {
			return properties, nil
		}
		logicalZero, zeroErr := context.Values().Zero(
			context,
			nil,
			pointee,
		)
		if zeroErr != nil {
			return nil, zeroErr
		}
		if logicalZero.Value() == nil {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: pointee.String(),
				Reason:   "reflection pointer zero is absent",
			}
		}
		if len(logicalZero.Before()) != 0 {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: pointee.String(),
				Reason:   "reflection pointer zero has prerequisites",
			}
		}
		allocatePointer, pointerErr := context.Names().TsonicCore(
			tsoniccore.SymbolAllocatePointer,
		)
		if pointerErr != nil {
			return nil, pointerErr
		}
		scaffold.requests = append(
			scaffold.requests,
			allocatePointer.Requests()...,
		)
		scaffold.requests = append(
			scaffold.requests,
			logicalZero.Requests()...,
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
							allocatePointer.Expression(factory),
							nil,
							nil,
							[]tsgo.Expression{logicalZero.Value()},
							tsgo.NodeFlagsNone,
						)},
					),
				),
			),
		))
	}
	return properties, nil
}
