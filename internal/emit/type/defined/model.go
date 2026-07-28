package defined

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	BrandMember = "$goType"
	ValueMember = "$value"
)

type Model struct {
	named      *types.Named
	typeName   *types.TypeName
	underlying *types.Basic
}

func Resolve(sourceType types.Type) (Model, bool) {
	if sourceType == nil {
		return Model{}, false
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil || named.TypeParams().Len() != 0 {
		return Model{}, false
	}
	underlying, ok := named.Underlying().(*types.Basic)
	if !ok {
		return Model{}, false
	}
	return Model{
		named:      named,
		typeName:   named.Obj(),
		underlying: underlying,
	}, true
}

func (m Model) Type() *types.Named {
	return m.named
}

func (m Model) TypeName() *types.TypeName {
	return m.typeName
}

func (m Model) Underlying() *types.Basic {
	return m.underlying
}

func (m Model) Unwrap(
	factory tsgo.Factory,
	value tsgo.Expression,
) tsgo.Expression {
	return factory.PropertyAccessExpression(
		value,
		nil,
		factory.Identifier(ValueMember),
		tsgo.NodeFlagsNone,
	)
}

func (m Model) Construct(
	context api.Context,
	value tsgo.Expression,
	requests ...api.RootRequest,
) (api.ExpressionEmission, error) {
	return m.Wrap(
		context,
		api.DirectExpression(value, requests...),
	)
}

func (m Model) Wrap(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().Reference(m.typeName)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		value.Before(),
		context.Factory().NewExpression(
			context.Factory().Identifier(reference.Name()),
			nil,
			[]tsgo.Expression{value.Value()},
		),
		api.CombineRequests(value.Requests(), reference.Requests()),
	)
}
