package frontend

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
)

type typeBuilder struct {
	objects    *objectIndex
	byGoType   map[types.Type]identity.SemanticTypeID
	records    map[identity.SemanticTypeID]semantic.Type
	building   map[types.Type]bool
	pending    map[types.Type]bool
	completing map[types.Type]bool
	work       int
}

func newTypeBuilder(objects *objectIndex) *typeBuilder {
	builder := &typeBuilder{
		objects:    objects,
		byGoType:   map[types.Type]identity.SemanticTypeID{},
		records:    map[identity.SemanticTypeID]semantic.Type{},
		building:   map[types.Type]bool{},
		pending:    map[types.Type]bool{},
		completing: map[types.Type]bool{},
	}
	objects.typeBuilder = builder
	return builder
}

func (builder *typeBuilder) build(
	typ types.Type,
) (identity.SemanticTypeID, error) {
	if typ == nil {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"semantic type materialization received nil",
		)
	}
	switch typed := typ.(type) {
	case *types.Named:
		return builder.buildNamed(typed)
	case *types.Alias:
		return builder.buildAlias(typed)
	case *types.TypeParam:
		return builder.buildTypeParameter(typed)
	}
	if existing := builder.byGoType[typ]; !existing.IsZero() {
		return existing, nil
	}
	builder.work++
	if builder.building[typ] {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"non-nominal semantic type cycle at %T", typ,
		)
	}
	builder.building[typ] = true
	spec, err := builder.typeSpec(typ)
	if err != nil {
		delete(builder.building, typ)
		return identity.SemanticTypeID{}, fmt.Errorf(
			"materialize Go type %T %q: %w",
			typ, types.TypeString(typ, nil), err,
		)
	}
	record, err := semantic.NewType(spec)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	delete(builder.building, typ)
	if err := builder.admit(typ, record); err != nil {
		return identity.SemanticTypeID{}, err
	}
	return record.ID(), nil
}

func (builder *typeBuilder) reference(
	typ types.Type,
) (identity.SemanticTypeID, error) {
	switch typed := typ.(type) {
	case *types.Named:
		return builder.namedID(typed)
	case *types.Alias:
		return builder.aliasID(typed)
	case *types.TypeParam:
		return builder.typeParameterID(typed)
	default:
		return builder.build(typ)
	}
}

func (builder *typeBuilder) buildNamed(
	typed *types.Named,
) (identity.SemanticTypeID, error) {
	nominal, err := builder.namedID(typed)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if _, complete := builder.records[nominal]; complete {
		builder.byGoType[typed] = nominal
		delete(builder.pending, typed)
		return nominal, nil
	}
	if builder.completing[typed] {
		return nominal, nil
	}
	builder.completing[typed] = true
	defer delete(builder.completing, typed)
	underlying, err := builder.reference(typed.Underlying())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	methods, err := builder.namedMethods(typed)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	declaration, err := builder.objects.declarationID(typed.Obj())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	arguments, err := builder.typeList(typed.TypeArgs())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	record, err := semantic.NewType(semantic.TypeSpec{
		Kind:        semantic.TypeNamed,
		Declaration: declaration,
		Arguments:   arguments,
		Underlying:  underlying,
		Methods:     methods,
	})
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if record.ID() != nominal {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"named type identity changed while materializing %s",
			declaration,
		)
	}
	if err := builder.admit(typed, record); err != nil {
		return identity.SemanticTypeID{}, err
	}
	delete(builder.pending, typed)
	return nominal, nil
}

func (builder *typeBuilder) namedID(
	typed *types.Named,
) (identity.SemanticTypeID, error) {
	if existing := builder.byGoType[typed]; !existing.IsZero() {
		return existing, nil
	}
	declaration, err := builder.objects.declarationID(typed.Obj())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	arguments, err := builder.typeList(typed.TypeArgs())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	nominal, err := semantic.NominalTypeID(
		semantic.TypeNamed,
		declaration,
		arguments,
	)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	builder.byGoType[typed] = nominal
	builder.pending[typed] = true
	return nominal, nil
}

func (builder *typeBuilder) buildAlias(
	typed *types.Alias,
) (identity.SemanticTypeID, error) {
	nominal, err := builder.aliasID(typed)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if _, complete := builder.records[nominal]; complete {
		builder.byGoType[typed] = nominal
		delete(builder.pending, typed)
		return nominal, nil
	}
	if builder.completing[typed] {
		return nominal, nil
	}
	builder.completing[typed] = true
	defer delete(builder.completing, typed)
	declaration, err := builder.objects.declarationID(typed.Obj())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	arguments, err := builder.typeList(typed.TypeArgs())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	target, err := builder.reference(types.Unalias(typed))
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	record, err := semantic.NewType(semantic.TypeSpec{
		Kind:        semantic.TypeAlias,
		Declaration: declaration,
		Arguments:   arguments,
		Target:      target,
	})
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if record.ID() != nominal {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"alias identity changed while materializing %s",
			declaration,
		)
	}
	if err := builder.admit(typed, record); err != nil {
		return identity.SemanticTypeID{}, err
	}
	delete(builder.pending, typed)
	return nominal, nil
}

func (builder *typeBuilder) aliasID(
	typed *types.Alias,
) (identity.SemanticTypeID, error) {
	if existing := builder.byGoType[typed]; !existing.IsZero() {
		return existing, nil
	}
	declaration, err := builder.objects.declarationID(typed.Obj())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	arguments, err := builder.typeList(typed.TypeArgs())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	nominal, err := semantic.NominalTypeID(
		semantic.TypeAlias,
		declaration,
		arguments,
	)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	builder.byGoType[typed] = nominal
	builder.pending[typed] = true
	return nominal, nil
}

func (builder *typeBuilder) buildTypeParameter(
	typed *types.TypeParam,
) (identity.SemanticTypeID, error) {
	nominal, err := builder.typeParameterID(typed)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if _, complete := builder.records[nominal]; complete {
		builder.byGoType[typed] = nominal
		delete(builder.pending, typed)
		return nominal, nil
	}
	if builder.completing[typed] {
		return nominal, nil
	}
	builder.completing[typed] = true
	defer delete(builder.completing, typed)
	owner, present := builder.objects.typeParameterOwner(typed)
	if !present {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"type parameter %s has no canonical generic owner (materialized-package=%s, object-package=%v, parent=%v)",
			typed.Obj().Name(),
			builder.objects.input.id,
			typed.Obj().Pkg(),
			typed.Obj().Parent(),
		)
	}
	constraint, err := builder.reference(typed.Constraint())
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	record, err := semantic.NewType(semantic.TypeSpec{
		Kind:       semantic.TypeParameter,
		Parameter:  owner,
		Constraint: constraint,
	})
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	if record.ID() != nominal {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"type-parameter identity changed at %s", owner,
		)
	}
	if err := builder.admit(typed, record); err != nil {
		return identity.SemanticTypeID{}, err
	}
	delete(builder.pending, typed)
	return nominal, nil
}

func (builder *typeBuilder) typeParameterID(
	typed *types.TypeParam,
) (identity.SemanticTypeID, error) {
	if existing := builder.byGoType[typed]; !existing.IsZero() {
		return existing, nil
	}
	owner, present := builder.objects.typeParameterOwner(typed)
	if !present {
		return identity.SemanticTypeID{}, fmt.Errorf(
			"type parameter %s has no canonical generic owner (materialized-package=%s, object-package=%v, parent=%v)",
			typed.Obj().Name(),
			builder.objects.input.id,
			typed.Obj().Pkg(),
			typed.Obj().Parent(),
		)
	}
	nominal, err := semantic.TypeParameterTypeID(owner)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	builder.byGoType[typed] = nominal
	builder.pending[typed] = true
	return nominal, nil
}

func (builder *typeBuilder) typeSpec(
	typ types.Type,
) (semantic.TypeSpec, error) {
	switch typed := typ.(type) {
	case *types.Basic:
		kind, err := semanticBasic(typed.Kind())
		return semantic.TypeSpec{
			Kind: semantic.TypeBasic, Basic: kind,
		}, err
	case *types.Pointer:
		return builder.elementSpec(semantic.TypePointer, typed.Elem())
	case *types.Slice:
		return builder.elementSpec(semantic.TypeSlice, typed.Elem())
	case *types.Array:
		element, err := builder.reference(typed.Elem())
		return semantic.TypeSpec{
			Kind: semantic.TypeArray, Element: element,
			Length: typed.Len(),
		}, err
	case *types.Map:
		key, err := builder.reference(typed.Key())
		if err != nil {
			return semantic.TypeSpec{}, err
		}
		element, err := builder.reference(typed.Elem())
		return semantic.TypeSpec{
			Kind: semantic.TypeMap, Key: key, Element: element,
		}, err
	case *types.Chan:
		element, err := builder.reference(typed.Elem())
		return semantic.TypeSpec{
			Kind: semantic.TypeChannel, Element: element,
			Direction: semanticChannelDirection(typed.Dir()),
		}, err
	case *types.Signature:
		signature, err := builder.signature(typed)
		return semantic.TypeSpec{
			Kind: semantic.TypeSignature, Signature: signature,
		}, err
	case *types.Struct:
		fields, err := builder.structFields(typed)
		return semantic.TypeSpec{
			Kind: semantic.TypeStruct, Fields: fields,
		}, err
	case *types.Interface:
		return builder.interfaceSpec(typed)
	case *types.Tuple:
		elements, err := builder.tuple(typed)
		return semantic.TypeSpec{
			Kind: semantic.TypeTuple, Elements: elements,
		}, err
	case *types.Union:
		terms, err := builder.unionTerms(typed)
		return semantic.TypeSpec{
			Kind: semantic.TypeUnion, Terms: terms,
		}, err
	default:
		return semantic.TypeSpec{}, fmt.Errorf(
			"unsupported go/types type %T", typ,
		)
	}
}

func (builder *typeBuilder) elementSpec(
	kind semantic.TypeKind,
	elementType types.Type,
) (semantic.TypeSpec, error) {
	element, err := builder.reference(elementType)
	return semantic.TypeSpec{
		Kind: kind, Element: element,
	}, err
}
