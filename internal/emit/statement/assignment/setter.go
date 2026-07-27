package assignment

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitSetter(
	context api.Context,
	target api.StoreTargetEmission,
	value api.ExpressionEmission,
) (api.StatementEmission, error) {
	operands := []api.ExpressionEmission{target.SetterReceiver()}
	operands = append(operands, target.SetterArguments()...)
	operands = append(operands, value)
	values, statements, requests, err := arrangeSetterOperands(
		context,
		operands,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	arguments := values[1:]
	call := context.Factory().CallExpression(
		context.Factory().PropertyAccessExpression(
			values[0],
			nil,
			context.Factory().Identifier(target.SetterMember()),
			tsgo.NodeFlagsNone,
		),
		nil,
		nil,
		arguments,
		tsgo.NodeFlagsNone,
	)
	statements = append(
		statements,
		context.Factory().ExpressionStatement(call),
	)
	return api.NewStatementEmission(statements, requests)
}

func arrangeSetterOperands(
	context api.Context,
	operands []api.ExpressionEmission,
) (
	[]tsgo.Expression,
	[]tsgo.Statement,
	[]api.PlacementRequest,
	error,
) {
	capture := make([]bool, len(operands))
	laterHasPrerequisites := false
	for index := len(operands) - 1; index >= 0; index-- {
		capture[index] = laterHasPrerequisites
		if len(operands[index].Before()) != 0 {
			laterHasPrerequisites = true
		}
	}
	values := make([]tsgo.Expression, 0, len(operands))
	var statements []tsgo.Statement
	var requests []api.PlacementRequest
	for index, operand := range operands {
		statements = append(statements, operand.Before()...)
		value := operand.Value()
		if capture[index] {
			name, err := context.Names().Temporary(
				api.TemporaryStoreOperand,
			)
			if err != nil {
				return nil, nil, nil, err
			}
			statements = append(
				statements,
				variableStatement(
					context,
					tsgo.NodeFlagsConst,
					name,
					value,
				),
			)
			value = context.Factory().Identifier(name)
		}
		values = append(values, value)
		requests = append(requests, operand.Requests()...)
	}
	return values, statements, requests, nil
}
