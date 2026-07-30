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
	if scalarKey && representedBasic(context, source.Elem()) {
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
