package assignment

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
)

func emitSetter(
	context api.Context,
	target api.StoreTargetEmission,
	value api.ExpressionEmission,
) (api.StatementEmission, error) {
	call, err := target.AccessorStore(context, value)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(
		call.Before(),
		context.Factory().ExpressionStatement(call.Value()),
	)
	return api.NewStatementEmission(statements, call.Requests())
}
