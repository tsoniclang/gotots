package defined

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	BrandMember    = "$goType"
	ValueMember    = "$value"
	FromMember     = "$from"
	ValueOfMember  = "$valueOf"
	MapReadMember  = "$readMap"
	MapStoreMember = "$storeMap"
	MapWrapMember  = "$wrapMap"
)

type Model struct {
	named      *types.Named
	typeName   *types.TypeName
	underlying types.Type
	family     Family
}

type Family uint8

const (
	FamilyInvalid Family = iota
	FamilyBasic
	FamilyArray
	FamilySlice
	FamilyPointer
	FamilyCallable
	FamilyMap
	FamilyChannel
)

func Resolve(sourceType types.Type) (Model, bool) {
	if sourceType == nil {
		return Model{}, false
	}
	named, ok := types.Unalias(sourceType).(*types.Named)
	if !ok || named.Obj() == nil {
		return Model{}, false
	}
	underlying := named.Underlying()
	var family Family
	switch underlying.(type) {
	case *types.Basic:
		family = FamilyBasic
	case *types.Array:
		family = FamilyArray
	case *types.Slice:
		family = FamilySlice
	case *types.Pointer:
		family = FamilyPointer
	case *types.Signature:
		signature := underlying.(*types.Signature)
		if !callable.Supports(signature) || signature.Recv() != nil {
			return Model{}, false
		}
		family = FamilyCallable
	case *types.Map:
		family = FamilyMap
	case *types.Chan:
		family = FamilyChannel
	default:
		return Model{}, false
	}
	if named.TypeParams().Len() != 0 &&
		named != named.Origin() &&
		named.TypeArgs().Len() != named.TypeParams().Len() {
		return Model{}, false
	}
	return Model{
		named:      named,
		typeName:   named.Obj(),
		underlying: underlying,
		family:     family,
	}, true
}

func ResolveBasic(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilyBasic
}

func ResolveArray(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilyArray
}

func ResolveSlice(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilySlice
}

func ResolvePointer(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilyPointer
}

func ResolveCallable(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilyCallable
}

func ResolveMap(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilyMap
}

func ResolveChannel(sourceType types.Type) (Model, bool) {
	model, ok := Resolve(sourceType)
	return model, ok && model.family == FamilyChannel
}

func (m Model) Type() *types.Named {
	return m.named
}

func (m Model) TypeName() *types.TypeName {
	return m.typeName
}

func (m Model) Underlying() types.Type {
	return m.underlying
}

func (m Model) Family() Family {
	return m.family
}

func (m Model) NilCapable() bool {
	return m.family == FamilySlice ||
		m.family == FamilyPointer ||
		m.family == FamilyCallable ||
		m.family == FamilyMap ||
		m.family == FamilyChannel
}

func (m Model) Basic() (*types.Basic, bool) {
	basic, ok := m.underlying.(*types.Basic)
	return basic, ok && m.family == FamilyBasic
}

func (m Model) Array() (*types.Array, bool) {
	array, ok := m.underlying.(*types.Array)
	return array, ok && m.family == FamilyArray
}

func (m Model) Slice() (*types.Slice, bool) {
	slice, ok := m.underlying.(*types.Slice)
	return slice, ok && m.family == FamilySlice
}

func (m Model) Pointer() (*types.Pointer, bool) {
	pointer, ok := m.underlying.(*types.Pointer)
	return pointer, ok && m.family == FamilyPointer
}

func (m Model) Callable() (*types.Signature, bool) {
	signature, ok := m.underlying.(*types.Signature)
	return signature, ok && m.family == FamilyCallable
}

func (m Model) Map() (*types.Map, bool) {
	mapType, ok := m.underlying.(*types.Map)
	return mapType, ok && m.family == FamilyMap
}

func (m Model) Channel() (*types.Chan, bool) {
	channel, ok := m.underlying.(*types.Chan)
	return channel, ok && m.family == FamilyChannel
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

func (m Model) Project(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	switch m.family {
	case FamilySlice, FamilyPointer, FamilyCallable, FamilyChannel:
		reference, err := context.Names().Reference(m.typeName)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		return api.NewExpressionEmission(
			value.Before(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(ValueOfMember),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{value.Value()},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(value.Requests(), reference.Requests()),
		)
	default:
		return api.NewExpressionEmission(
			value.Before(),
			m.Unwrap(context.Factory(), value.Value()),
			value.Requests(),
		)
	}
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
	switch m.family {
	case FamilySlice, FamilyPointer, FamilyCallable, FamilyChannel:
		return api.NewExpressionEmission(
			value.Before(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(FromMember),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{value.Value()},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(value.Requests(), reference.Requests()),
		)
	case FamilyMap:
		return api.NewExpressionEmission(
			value.Before(),
			context.Factory().CallExpression(
				context.Factory().PropertyAccessExpression(
					context.Factory().Identifier(reference.Name()),
					nil,
					context.Factory().Identifier(MapWrapMember),
					tsgo.NodeFlagsNone,
				),
				nil,
				nil,
				[]tsgo.Expression{value.Value()},
				tsgo.NodeFlagsNone,
			),
			api.CombineRequests(value.Requests(), reference.Requests()),
		)
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
