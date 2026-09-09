package namedstruct

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	pointermarker "github.com/tsoniclang/gotots/internal/emit/marker/pointer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func storageSchema(
	context api.Context,
	name string,
	fields []layoutField,
	moduleExport bool,
) ([]tsgo.Statement, []api.RootRequest, error) {
	factory := context.Factory()
	members := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
	var requests []api.RootRequest
	for _, selected := range fields {
		value, err := pointermarker.Operation(context, tsoniccore.SymbolField,
			[]api.TypeEmission{api.DirectType(selected.storageType)}, nil)
		if err != nil {
			return nil, nil, err
		}
		members = append(members, factory.PropertyAssignment(
			nil, factory.Identifier(selected.field.name), nil, selected.storageType, value.Value(),
		))
		requests = append(requests, value.Requests()...)
	}
	shape, err := pointermarker.Operation(context, tsoniccore.SymbolStruct, nil,
		[]api.ExpressionEmission{api.DirectExpression(factory.ObjectLiteralExpression(members, true), requests...)})
	if err != nil {
		return nil, nil, err
	}
	var modifiers []tsgo.ModifierLike
	if moduleExport {
		modifiers = []tsgo.ModifierLike{factory.ExportKeyword()}
	}
	return []tsgo.Statement{
		factory.VariableStatement(modifiers, factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(factory.Identifier(name), nil, storageShape(context, fields), shape.Value())}, tsgo.NodeFlagsConst)),
		factory.TypeAliasDeclaration(modifiers, factory.Identifier(name), nil,
			factory.TypeQueryNode(factory.Identifier(name), nil)),
	}, shape.Requests(), nil
}
