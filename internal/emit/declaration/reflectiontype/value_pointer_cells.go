package reflectiontype

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	pointertype "github.com/tsoniclang/gotots/internal/emit/type/pointer"
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
	representation, err := pointertype.Observe(
		context,
		types.NewPointer(pointee),
		api.PointerRepresentationDemandNone,
	)
	if err != nil {
		return nil, err
	}
	cellValue := memberAccess(factory, "instance", "value")
	cellRead, err := context.ContainerStorage().FromPointerStorage(
		context,
		nil,
		pointee,
		representation,
		api.DirectExpression(cellValue),
	)
	if err != nil {
		return nil, err
	}
	cellWrite, err := context.ContainerStorage().ToPointerStorage(
		context,
		nil,
		pointee,
		representation,
		api.DirectExpression(guardedForeignPayload(
			scaffold,
			sliceAdapter,
			"Value.Set",
		)),
	)
	if err != nil {
		return nil, err
	}
	if len(cellRead.Before()) != 0 || len(cellWrite.Before()) != 0 {
		return nil, &api.GeneratedArtifactShapeError{
			Artifact: pointee.String(),
			Reason:   "reflection pointer cell conversion has prerequisites",
		}
	}
	scaffold.requests = append(scaffold.requests, cellRead.Requests()...)
	scaffold.requests = append(scaffold.requests, cellWrite.Requests()...)
	location := locationLiteral(scaffold, locationCallbacks{
		descriptor: descriptor,
		settable:   true,
		get: factory.NewExpression(
			sliceAdapter.Expression(factory),
			nil,
			[]tsgo.Expression{cellRead.Value()},
		),
		set: factory.Block([]tsgo.Statement{
			factory.ExpressionStatement(factory.BinaryExpression(
				nil,
				cellValue,
				nil,
				factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
				cellWrite.Value(),
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
		storedZero, storageErr := context.ContainerStorage().ToPointerStorage(
			context,
			nil,
			pointee,
			representation,
			logicalZero,
		)
		if storageErr != nil {
			return nil, storageErr
		}
		if len(storedZero.Before()) != 0 {
			return nil, &api.GeneratedArtifactShapeError{
				Artifact: pointee.String(),
				Reason:   "reflection pointer zero has prerequisites",
			}
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
			storedZero.Requests()...,
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
							[]tsgo.Expression{storedZero.Value()},
							tsgo.NodeFlagsNone,
						)},
					),
				),
			),
		))
	}
	return properties, nil
}
