package semantic

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type objectReferenceRef uint64
type constantRef uint64
type selectionRef uint64
type instanceRef uint64

type occurrenceRefRange struct {
	start uint64
	count uint64
}

type definitionRefRange struct {
	start uint64
	count uint64
}

type implicitOperationRange struct {
	start uint64
	count uint64
}

type integerRange struct {
	start uint64
	count uint64
}

type typeRefRange struct {
	start uint64
	count uint64
}

type storedOperation struct {
	id            operationRef
	kind          OperationKind
	syntax        catalog.Kind
	variant       catalog.Variant
	role          catalog.Role
	token         catalog.TokenKind
	mode          ValueMode
	arity         ResultArity
	place         PlaceKind
	resultType    typeRef
	expectedType  typeRef
	addressable   bool
	assignable    bool
	hasOk         bool
	constant      constantRef
	object        objectReferenceRef
	selection     selectionRef
	instance      instanceRef
	operands      occurrenceRefRange
	definitions   definitionRefRange
	implicit      implicitOperationRange
	controlTarget operationRef
	label         bindingRef
}

type storedObjectReference struct {
	kind        ObjectReferenceKind
	declaration declarationRef
	binding     bindingRef
}

type storedConstant struct {
	kind  ConstantKind
	exact string
}

type storedSelection struct {
	kind     SelectionKind
	receiver typeRef
	object   declarationRef
	index    integerRange
	indirect bool
}

type storedInstance struct {
	target    objectReferenceRef
	types     typeRefRange
	signature typeRef
}

type storedImplicitOperation struct {
	kind    catalog.ImplicitOp
	site    occurrenceRef
	ordinal int
	source  typeRef
	target  typeRef
}

type packageOperationBuilder struct {
	records          []storedOperation
	objects          []storedObjectReference
	constants        []storedConstant
	selections       []storedSelection
	instances        []storedInstance
	operands         []occurrenceRef
	definitions      []definitionRef
	implicit         []storedImplicitOperation
	selectionIndexes []int
	instanceTypes    []typeRef
}

func (builder *packageOperationBuilder) add(
	identities *packageIdentityBuilder,
	record Operation,
) {
	spec := record.spec
	stored := storedOperation{
		id:            identities.operation(spec.ID),
		kind:          spec.Kind,
		syntax:        spec.Syntax,
		variant:       spec.Variant,
		role:          spec.Role,
		token:         spec.Token,
		mode:          spec.Mode,
		arity:         spec.Arity,
		place:         spec.Place,
		resultType:    identities.typeID(spec.ResultType),
		expectedType:  identities.typeID(spec.ExpectedType),
		addressable:   spec.Addressable,
		assignable:    spec.Assignable,
		hasOk:         spec.HasOk,
		constant:      builder.addConstant(spec.Constant),
		object:        builder.addObject(identities, spec.Object),
		selection:     builder.addSelection(identities, spec.Selection),
		instance:      builder.addInstance(identities, spec.Instance),
		operands:      builder.addOperands(identities, spec.Operands),
		definitions:   builder.addDefinitions(identities, spec.Definitions),
		implicit:      builder.addImplicit(identities, spec.Implicit),
		controlTarget: identities.operation(spec.ControlTarget),
		label:         identities.binding(spec.Label),
	}
	builder.records = append(builder.records, stored)
}

func (builder *packageOperationBuilder) addConstant(
	constant Constant,
) constantRef {
	if constant.IsZero() {
		return 0
	}
	builder.constants = append(builder.constants, storedConstant{
		kind: constant.kind, exact: constant.exact,
	})
	return constantRef(len(builder.constants))
}

func (builder *packageOperationBuilder) addObject(
	identities *packageIdentityBuilder,
	object ObjectReference,
) objectReferenceRef {
	if object.kind == ObjectReferenceNone {
		return 0
	}
	builder.objects = append(builder.objects, storedObjectReference{
		kind: object.kind,
		declaration: identities.declaration(
			object.declaration,
		),
		binding: identities.binding(object.binding),
	})
	return objectReferenceRef(len(builder.objects))
}

func (builder *packageOperationBuilder) addSelection(
	identities *packageIdentityBuilder,
	selection Selection,
) selectionRef {
	if selection.IsZero() {
		return 0
	}
	index := integerRange{
		start: uint64(len(builder.selectionIndexes)),
		count: uint64(len(selection.index)),
	}
	builder.selectionIndexes = append(
		builder.selectionIndexes, selection.index...,
	)
	builder.selections = append(builder.selections, storedSelection{
		kind:     selection.kind,
		receiver: identities.typeID(selection.receiver),
		object:   identities.declaration(selection.object),
		index:    index, indirect: selection.indirect,
	})
	return selectionRef(len(builder.selections))
}

func (builder *packageOperationBuilder) addInstance(
	identities *packageIdentityBuilder,
	instance Instance,
) instanceRef {
	if instance.IsZero() {
		return 0
	}
	types := typeRefRange{
		start: uint64(len(builder.instanceTypes)),
		count: uint64(len(instance.types)),
	}
	for _, typeID := range instance.types {
		builder.instanceTypes = append(
			builder.instanceTypes,
			identities.typeID(typeID),
		)
	}
	builder.instances = append(builder.instances, storedInstance{
		target:    builder.addObject(identities, instance.target),
		types:     types,
		signature: identities.typeID(instance.signature),
	})
	return instanceRef(len(builder.instances))
}

func (builder *packageOperationBuilder) addOperands(
	identities *packageIdentityBuilder,
	values []identity.OccurrenceID,
) occurrenceRefRange {
	out := occurrenceRefRange{
		start: uint64(len(builder.operands)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.operands = append(
			builder.operands,
			identities.occurrence(value),
		)
	}
	return out
}

func (builder *packageOperationBuilder) addDefinitions(
	identities *packageIdentityBuilder,
	values []identity.DefinitionID,
) definitionRefRange {
	out := definitionRefRange{
		start: uint64(len(builder.definitions)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.definitions = append(
			builder.definitions,
			identities.definition(value),
		)
	}
	return out
}

func (builder *packageOperationBuilder) addImplicit(
	identities *packageIdentityBuilder,
	values []ImplicitOperation,
) implicitOperationRange {
	out := implicitOperationRange{
		start: uint64(len(builder.implicit)),
		count: uint64(len(values)),
	}
	for _, value := range values {
		builder.implicit = append(
			builder.implicit,
			storedImplicitOperation{
				kind:    value.kind,
				site:    identities.occurrence(value.site),
				ordinal: value.ordinal,
				source:  identities.typeID(value.source),
				target:  identities.typeID(value.target),
			},
		)
	}
	return out
}

type packageOperationStore struct {
	records          []storedOperation
	objects          []storedObjectReference
	constants        []storedConstant
	selections       []storedSelection
	instances        []storedInstance
	operands         []occurrenceRef
	definitions      []definitionRef
	implicit         []storedImplicitOperation
	selectionIndexes []int
	instanceTypes    []typeRef
}

func (builder *packageOperationBuilder) seal(
	remap packageIdentityRemap,
) (packageOperationStore, error) {
	store := packageOperationStore{
		records: builder.records, objects: builder.objects,
		constants: builder.constants, selections: builder.selections,
		instances: builder.instances, operands: builder.operands,
		definitions: builder.definitions, implicit: builder.implicit,
		selectionIndexes: builder.selectionIndexes,
		instanceTypes:    builder.instanceTypes,
	}
	if err := store.remap(remap); err != nil {
		return packageOperationStore{}, err
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].id < store.records[right].id
	})
	return store, nil
}

func (store *packageOperationStore) remap(
	remap packageIdentityRemap,
) error {
	var err error
	for index := range store.records {
		record := &store.records[index]
		if record.id, err = remapReference(
			record.id, remap.operations,
		); err != nil {
			return err
		}
		if record.resultType, err = remapReference(
			record.resultType, remap.types,
		); err != nil {
			return err
		}
		if record.expectedType, err = remapReference(
			record.expectedType, remap.types,
		); err != nil {
			return err
		}
		if record.controlTarget, err = remapReference(
			record.controlTarget, remap.operations,
		); err != nil {
			return err
		}
		if record.label, err = remapReference(
			record.label, remap.bindings,
		); err != nil {
			return err
		}
	}
	for index := range store.objects {
		object := &store.objects[index]
		if object.declaration, err = remapReference(
			object.declaration, remap.declarations,
		); err != nil {
			return err
		}
		if object.binding, err = remapReference(
			object.binding, remap.bindings,
		); err != nil {
			return err
		}
	}
	for index := range store.selections {
		selection := &store.selections[index]
		if selection.receiver, err = remapReference(
			selection.receiver, remap.types,
		); err != nil {
			return err
		}
		if selection.object, err = remapReference(
			selection.object, remap.declarations,
		); err != nil {
			return err
		}
	}
	for index := range store.instances {
		instance := &store.instances[index]
		if instance.signature, err = remapReference(
			instance.signature, remap.types,
		); err != nil {
			return err
		}
	}
	for index := range store.implicit {
		implicit := &store.implicit[index]
		if implicit.site, err = remapReference(
			implicit.site, remap.occurrences,
		); err != nil {
			return err
		}
		if implicit.source, err = remapReference(
			implicit.source, remap.types,
		); err != nil {
			return err
		}
		if implicit.target, err = remapReference(
			implicit.target, remap.types,
		); err != nil {
			return err
		}
	}
	if err := remapReferences(store.operands, remap.occurrences); err != nil {
		return err
	}
	if err := remapReferences(store.definitions, remap.definitions); err != nil {
		return err
	}
	return remapReferences(store.instanceTypes, remap.types)
}
