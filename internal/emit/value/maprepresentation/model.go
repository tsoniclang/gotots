package maprepresentation

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
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
	source ast.Node,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	zero, err := context.Values().Zero(
		context.WithRole(api.RoleMapReceiver),
		source,
		m.defined.Underlying(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return m.staticMapOperation(
		context,
		definedtype.MapReadMember,
		value,
		zero,
	)
}

func (m Model) StoreReceiver(
	context api.Context,
	_ ast.Node,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	return m.staticMapOperation(context, definedtype.MapStoreMember, value)
}

func (m Model) WrapConverted(
	context api.Context,
	value api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	if !m.nominal {
		return value, nil
	}
	return m.staticMapOperation(context, definedtype.MapWrapMember, value)
}

func (m Model) staticMapOperation(
	context api.Context,
	member string,
	value api.ExpressionEmission,
	arguments ...api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	reference, err := context.Names().Reference(m.TypeName())
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	values := []tsgo.Expression{value.Value()}
	requests := value.Requests()
	before := value.Before()
	for _, argument := range arguments {
		before = append(before, argument.Before()...)
		values = append(values, argument.Value())
		requests = append(requests, argument.Requests()...)
	}
	return api.NewExpressionEmission(
		before,
		context.Factory().CallExpression(
			context.Factory().PropertyAccessExpression(
				context.Factory().Identifier(reference.Name()),
				nil,
				context.Factory().Identifier(member),
				tsgo.NodeFlagsNone,
			),
			nil,
			nil,
			values,
			tsgo.NodeFlagsNone,
		),
		api.CombineRequests(requests, reference.Requests()),
	)
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
