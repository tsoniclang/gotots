package operands

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Pair struct {
	before []tsgo.Statement
	left   api.ExpressionEmission
	right  api.ExpressionEmission
}

func Preserve(
	context api.Context,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
	temporaryKind api.TemporaryKind,
) (Pair, error) {
	before := left.Before()
	leftValue := left.Value()
	if len(right.Before()) != 0 {
		temporaryName, err := context.Names().Temporary(temporaryKind)
		if err != nil {
			return Pair{}, err
		}
		before = append(
			before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(temporaryName),
							nil,
							nil,
							leftValue,
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		leftValue = context.Factory().Identifier(temporaryName)
	}
	before = append(before, right.Before()...)
	return Pair{
		before: before,
		left: api.DirectExpression(
			leftValue,
			left.Requests()...,
		),
		right: api.DirectExpression(
			right.Value(),
			right.Requests()...,
		),
	}, nil
}

func (p Pair) Before() []tsgo.Statement {
	return append([]tsgo.Statement(nil), p.before...)
}

func (p Pair) Left() api.ExpressionEmission {
	return p.left
}

func (p Pair) Right() api.ExpressionEmission {
	return p.right
}

func Finish(
	pair Pair,
	target api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	before := pair.Before()
	before = append(before, target.Before()...)
	return api.NewExpressionEmission(
		before,
		target.Value(),
		target.Requests(),
	)
}
