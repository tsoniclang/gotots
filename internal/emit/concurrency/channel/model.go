package channel

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

type Model struct {
	source  types.Type
	channel *types.Chan
	defined definedtype.Model
}

func Resolve(sourceType types.Type) (Model, bool) {
	if sourceType == nil {
		return Model{}, false
	}
	if channel, ok := types.Unalias(sourceType).(*types.Chan); ok {
		return Model{source: sourceType, channel: channel}, true
	}
	defined, ok := definedtype.ResolveChannel(sourceType)
	if !ok {
		return Model{}, false
	}
	channel, valid := defined.Channel()
	if !valid {
		return Model{}, false
	}
	return Model{
		source:  sourceType,
		channel: channel,
		defined: defined,
	}, true
}

func (m Model) Type() types.Type {
	return m.source
}

func (m Model) Underlying() *types.Chan {
	return m.channel
}

func (m Model) Element() types.Type {
	if m.channel == nil {
		return nil
	}
	return m.channel.Elem()
}

func (m Model) Direction() types.ChanDir {
	if m.channel == nil {
		return 0
	}
	return m.channel.Dir()
}

func (m Model) Project(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if m.defined.Type() == nil {
		return value, nil
	}
	return m.defined.Project(context, value)
}

func (m Model) Wrap(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if m.defined.Type() == nil {
		return value, nil
	}
	return m.defined.Wrap(context, value)
}
