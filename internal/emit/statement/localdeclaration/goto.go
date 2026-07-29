package localdeclaration

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func gotoLocalAssignment(
	context api.Context,
	children api.ChildEmitter,
	selected binding,
) ([]tsgo.Statement, []api.RootRequest, bool, error) {
	if !context.IsGotoLocal(selected.object) {
		return nil, nil, false, nil
	}
	name, storage := context.AddressableStorage().Name(
		context,
		selected.object,
	)
	value := selected.value
	var err error
	if storage {
		value, err = context.AddressableStorage().Cell(
			context,
			children,
			selected.sourceName,
			selected.object.Type(),
			value,
		)
	} else {
		name, err = context.Names().Declare(selected.object)
	}
	if err != nil {
		return nil, nil, true, err
	}
	statements := value.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(
			context.Factory().BinaryExpression(
				nil,
				context.Factory().Identifier(name),
				nil,
				context.Factory().BinaryOperatorToken(
					tsgo.BinaryOperatorEqualsToken,
				),
				value.Value(),
			),
		),
	)
	return statements, value.Requests(), true, nil
}
