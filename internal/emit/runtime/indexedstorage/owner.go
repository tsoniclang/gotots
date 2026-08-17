package indexedstorage

import (
	panicruntime "github.com/tsoniclang/gotots/internal/emit/runtime/panic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Element(
	factory tsgo.Factory,
	panicName string,
	values tsgo.Expression,
	index tsgo.Expression,
	targetType tsgo.TypeNode,
) tsgo.AsExpression {
	present := factory.BinaryExpression(
		nil,
		index,
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorInKeyword),
		values,
	)
	value := factory.ConditionalExpression(
		present,
		factory.QuestionToken(),
		factory.ElementAccessExpression(
			values,
			nil,
			index,
			tsgo.NodeFlagsNone,
		),
		factory.ColonToken(),
		panicruntime.Call(
			factory,
			panicName,
			factory.StringLiteral(
				"dense storage index is absent",
				tsgo.TokenFlagsNone,
			),
		),
	)
	return factory.AsExpression(value, targetType)
}
