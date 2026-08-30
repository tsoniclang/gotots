package namedstruct

import (
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func valueStructureAssertion(
	context api.Context,
	fields []layoutField,
) (tsgo.Statement, []api.RootRequest, error) {
	structReference, err := context.Names().TsonicCore(tsoniccore.SymbolStruct)
	if err != nil {
		return nil, nil, err
	}
	properties := make([]tsgo.ObjectLiteralElementLike, 0, len(fields))
	requests := structReference.Requests()
	var fieldReference api.NameReference
	for _, selected := range fields {
		if selected.field.blank {
			continue
		}
		if fieldReference.Name() == "" {
			fieldReference, err = context.Names().TsonicCore(tsoniccore.SymbolField)
			if err != nil {
				return nil, nil, err
			}
			requests = api.CombineRequests(requests, fieldReference.Requests())
		}
		properties = append(properties, context.Factory().PropertyAssignment(
			nil,
			context.Factory().Identifier(selected.field.name),
			nil,
			selected.logicalType,
			context.Factory().CallExpression(
				fieldReference.Expression(context.Factory()),
				nil,
				[]tsgo.TypeNode{selected.logicalType},
				nil,
				tsgo.NodeFlagsNone,
			),
		))
	}
	shape := context.Factory().ObjectLiteralExpression(properties, true)
	return context.Factory().ExpressionStatement(
		context.Factory().CallExpression(
			structReference.Expression(context.Factory()),
			nil,
			nil,
			[]tsgo.Expression{shape},
			tsgo.NodeFlagsNone,
		),
	), requests, nil
}
