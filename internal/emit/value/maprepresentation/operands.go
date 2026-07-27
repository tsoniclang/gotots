package maprepresentation

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func ArrangeOperands(
	context api.Context,
	operands []api.ExpressionEmission,
) (
	[]tsgo.Expression,
	[]tsgo.Statement,
	[]api.PlacementRequest,
	error,
) {
	capture := false
	for _, operand := range operands {
		if len(operand.Before()) != 0 {
			capture = true
			break
		}
	}
	values := make([]tsgo.Expression, 0, len(operands))
	var before []tsgo.Statement
	var requests []api.PlacementRequest
	for _, operand := range operands {
		requests = append(requests, operand.Requests()...)
		if !capture {
			values = append(values, operand.Value())
			continue
		}
		name, err := context.Names().Temporary(api.TemporaryMapOperand)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, operand.Before()...)
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(name),
							nil,
							nil,
							operand.Value(),
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		values = append(values, context.Factory().Identifier(name))
	}
	return values, before, requests, nil
}
