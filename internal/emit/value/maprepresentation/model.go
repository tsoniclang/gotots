package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

type Storage uint8

const (
	StorageInvalid Storage = iota
	StorageScalar
	StorageSpecialized
)

type Model struct {
	sourceType types.Type
	source     *types.Map
	defined    definedtype.Model
	nominal    bool
	storage    Storage
}

func Source(
	context api.Context,
	sourceType types.Type,
) (Model, bool) {
	var source *types.Map
	defined, nominal := definedtype.ResolveMap(sourceType)
	if nominal {
		source, _ = defined.Map()
	} else {
		source, _ = types.Unalias(sourceType).(*types.Map)
	}
	if source == nil || !types.Comparable(source.Key()) {
		return Model{}, false
	}
	_, scalarKey := directKey(context, source.Key())
	if !scalarKey && !context.Values().SupportsHash(context, source.Key()) {
		return Model{}, false
	}
	storage := StorageSpecialized
	if scalarKey &&
		types.Identical(source.Key(), storageKeyType(source.Key())) &&
		representedBasic(context, source.Elem()) {
		storage = StorageScalar
	}
	return Model{
		sourceType: sourceType,
		source:     source,
		defined:    defined,
		nominal:    nominal,
		storage:    storage,
	}, true
}

func (m Model) Type() types.Type {
	return m.sourceType
}

func (m Model) Map() *types.Map {
	return m.source
}

func (m Model) Key() types.Type {
	return m.source.Key()
}

func (m Model) Element() types.Type {
	return m.source.Elem()
}

func (m Model) StorageKey() types.Type {
	return storageKeyType(m.source.Key())
}

func (m Model) Storage() Storage {
	return m.storage
}

func (m Model) Nominal() bool {
	return m.nominal
}

func (m Model) TypeName() *types.TypeName {
	if !m.nominal {
		return nil
	}
	return m.defined.TypeName()
}

func (m Model) ReadReceiver(
	context api.Context,
	_ ast.Node,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	return m.defined.Project(context, value)
}

func (m Model) StoreReceiver(
	context api.Context,
	_ ast.Node,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	return m.defined.Project(context, value)
}

func (m Model) TransferKey(
	context api.Context,
	source ast.Expr,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if m.source == nil || source == nil {
		return api.ExpressionEmission{}, &api.InvariantError{
			Role:   context.Role(),
			Reason: "map key transfer input is invalid",
		}
	}
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil || !types.AssignableTo(sourceType, m.Key()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryExpression, source)
	}
	return context.Values().Transfer(
		context,
		source,
		sourceType,
		m.Key(),
		api.ValueTransferRepresentation,
		value,
	)
}

func (m Model) WrapConverted(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	return m.defined.Wrap(context, value)
}

func (m Model) Wrap(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	return m.defined.Wrap(context, value)
}
