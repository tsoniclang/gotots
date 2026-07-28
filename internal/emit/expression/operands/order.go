package operands

import (
	"slices"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Item struct {
	emission api.ExpressionEmission
	omitted  tsgo.Expression
	present  bool
}

func Present(emission api.ExpressionEmission) Item {
	if emission.Value() == nil {
		panic("ordered operand target is nil")
	}
	return Item{emission: emission, present: true}
}

func Omitted(value tsgo.Expression) Item {
	if value == nil {
		panic("omitted ordered operand target is nil")
	}
	return Item{omitted: value}
}

type Sequence struct {
	before   []tsgo.Statement
	values   []tsgo.Expression
	requests []api.RootRequest
}

func Preserve(
	context api.Context,
	temporaryKind api.TemporaryKind,
	items ...Item,
) (Sequence, error) {
	capture := false
	for _, item := range items {
		if item.present && len(item.emission.Before()) != 0 {
			capture = true
			break
		}
	}
	sequence := Sequence{
		values: make([]tsgo.Expression, 0, len(items)),
	}
	for _, item := range items {
		if !item.present {
			sequence.values = append(sequence.values, item.omitted)
			continue
		}
		sequence.requests = append(
			sequence.requests,
			item.emission.Requests()...,
		)
		if !capture {
			sequence.values = append(
				sequence.values,
				item.emission.Value(),
			)
			continue
		}
		name, err := context.Names().Temporary(temporaryKind)
		if err != nil {
			return Sequence{}, err
		}
		sequence.before = append(
			sequence.before,
			item.emission.Before()...,
		)
		sequence.before = append(
			sequence.before,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{
						context.Factory().VariableDeclaration(
							context.Factory().Identifier(name),
							nil,
							nil,
							item.emission.Value(),
						),
					},
					tsgo.NodeFlagsConst,
				),
			),
		)
		sequence.values = append(
			sequence.values,
			context.Factory().Identifier(name),
		)
	}
	return sequence, nil
}

func (s Sequence) Before() []tsgo.Statement {
	return slices.Clone(s.before)
}

func (s Sequence) Values() []tsgo.Expression {
	return slices.Clone(s.values)
}

func (s Sequence) Requests() []api.RootRequest {
	return slices.Clone(s.requests)
}

type Pair struct {
	sequence Sequence
	left     api.ExpressionEmission
	right    api.ExpressionEmission
}

func PreservePair(
	context api.Context,
	left api.ExpressionEmission,
	right api.ExpressionEmission,
	temporaryKind api.TemporaryKind,
) (Pair, error) {
	sequence, err := Preserve(
		context,
		temporaryKind,
		Present(left),
		Present(right),
	)
	if err != nil {
		return Pair{}, err
	}
	values := sequence.Values()
	return Pair{
		sequence: sequence,
		left: api.DirectExpression(
			values[0],
			left.Requests()...,
		),
		right: api.DirectExpression(
			values[1],
			right.Requests()...,
		),
	}, nil
}

func (p Pair) Before() []tsgo.Statement {
	return p.sequence.Before()
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
