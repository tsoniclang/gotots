package memory

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/emit/value/structconstruction"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func recordSchema(context api.Context, source ast.Node, structure *types.Struct, represented api.TypeEmission, physical bool) (api.TypeEmission, []tsgo.Statement, error) {
	if _, literal := represented.Value().(tsgo.TypeLiteralNode); !literal {
		return represented, nil, nil
	}
	name, err := context.Names().Temporary(api.TemporaryConversionOperand)
	if err != nil {
		return api.TypeEmission{}, nil, err
	}
	factory := context.Factory()
	var members []tsgo.ObjectLiteralElementLike
	var requests []api.RootRequest
	for index := range structure.NumFields() {
		field := structure.Field(index)
		fieldName, err := structconstruction.FieldName(context.Names(), field, index)
		if err != nil {
			return api.TypeEmission{}, nil, err
		}
		fieldType, err := bindingStorageType(context, source, field.Type(), physical)
		if err != nil {
			return api.TypeEmission{}, nil, err
		}
		declaration, err := pointermarker.Operation(context, tsoniccore.SymbolField, []api.TypeEmission{fieldType}, nil)
		if err != nil {
			return api.TypeEmission{}, nil, err
		}
		members = append(members, factory.PropertyAssignment(nil, factory.Identifier(fieldName), nil, fieldType.Value(), declaration.Value()))
		requests = append(requests, declaration.Requests()...)
	}
	shape, err := pointermarker.Operation(context, tsoniccore.SymbolStruct, nil,
		[]api.ExpressionEmission{api.DirectExpression(factory.ObjectLiteralExpression(members, true), requests...)})
	if err != nil {
		return api.TypeEmission{}, nil, err
	}
	declaration := factory.VariableStatement(nil, factory.VariableDeclarationList([]tsgo.VariableDeclaration{
		factory.VariableDeclaration(factory.Identifier(name), nil, represented.Value(), shape.Value()),
	}, tsgo.NodeFlagsConst))
	return api.DirectType(factory.TypeQueryNode(factory.Identifier(name), nil),
		api.CombineRequests(represented.Requests(), shape.Requests())...), []tsgo.Statement{declaration}, nil
}
