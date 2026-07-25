package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

type normalizedPackageReachability struct {
	pkg            Package
	index          normalizedPackageIndex
	declarations   []bool
	bindings       []bool
	types          []bool
	operations     []bool
	unsupported    []bool
	typeQueue      []typeRef
	operationQueue []operationRef
}

func validateNormalizedPackageReachability(
	pkg Package,
	index normalizedPackageIndex,
) error {
	reachable := normalizedPackageReachability{
		pkg: pkg, index: index,
		declarations: make(
			[]bool, len(pkg.identities.declarations)+1,
		),
		bindings: make([]bool, len(pkg.identities.bindings)+1),
		types:    make([]bool, len(pkg.identities.types)+1),
		operations: make(
			[]bool, len(pkg.identities.operations)+1,
		),
		unsupported: make(
			[]bool, len(pkg.identities.unsupported)+1,
		),
	}
	reachable.markDefinitionRoots()
	for _, record := range pkg.declarations.records {
		reachable.markDeclaration(record.id)
	}
	for _, record := range pkg.bindings.records {
		reachable.markBinding(record.id)
	}
	for _, record := range pkg.resolutions.records {
		reachable.markResolution(record)
	}
	for _, record := range pkg.operations.records {
		identityRecord := pkg.identities.operations[record.id-1]
		if identityRecord.occurrence == 0 {
			reachable.markOperation(record.id)
		}
	}
	reachable.drainOperations()
	reachable.drainTypes()
	return reachable.requireComplete()
}

func (reachable *normalizedPackageReachability) markDefinitionRoots() {
	store := reachable.pkg.definitions
	for _, record := range store.records {
		bindings, _ := storedRelation(
			store.bindingRelations,
			record.bindings.start,
			record.bindings.count,
		)
		for _, binding := range bindings {
			reachable.markBinding(binding)
		}
		switch record.form {
		case DefinitionFormCallable:
			payload := store.callable[record.payload-1]
			reachable.markDeclarationRange(payload.declarations)
			reachable.markType(payload.signature)
			reachable.markBinding(payload.receiver)
		case DefinitionFormInitializer:
			payload := store.initializers[record.payload-1]
			reachable.markDeclarationRange(payload.declarations)
		case DefinitionFormBodyless:
			payload := store.bodyless[record.payload-1]
			reachable.markDeclaration(payload.declaration)
			reachable.markType(payload.signature)
			reachable.markBinding(payload.receiver)
		case DefinitionFormSynthetic:
			payload := store.synthetic[record.payload-1]
			reachable.markDeclaration(payload.declaration)
			reachable.markType(payload.signature)
		}
	}
}

func (reachable *normalizedPackageReachability) markDeclarationRange(
	relation declarationRefRange,
) {
	declarations, _ := storedRelation(
		reachable.pkg.definitions.declarationRelations,
		relation.start,
		relation.count,
	)
	for _, declaration := range declarations {
		reachable.markDeclaration(declaration)
	}
}

func (reachable *normalizedPackageReachability) markResolution(
	record storedResolution,
) {
	store := reachable.pkg.resolutions
	switch record.kind {
	case ResolutionStructuralOnly:
		payload := store.structural[record.payload-1]
		reachable.markDeclaration(payload.declaration)
		reachable.markType(payload.typeID)
	case ResolutionDeclaration:
		reachable.markDeclaration(
			store.declarations[record.payload-1],
		)
	case ResolutionBinding:
		reachable.markBinding(store.bindings[record.payload-1])
	case ResolutionType:
		reachable.markType(store.types[record.payload-1])
	case ResolutionOperation:
		reachable.markOperation(store.operations[record.payload-1])
	case ResolutionUnsupported:
		reachable.markUnsupported(
			store.unsupported[record.payload-1],
		)
	}
}

func (reachable *normalizedPackageReachability) markDeclaration(
	reference declarationRef,
) {
	if reference == 0 ||
		uint64(reference) >= uint64(len(reachable.declarations)) ||
		reachable.declarations[reference] {
		return
	}
	identityRecord :=
		reachable.pkg.identities.declarations[reference-1]
	if identityRecord.form ==
		identity.SemanticDeclarationMember {
		reachable.markType(identityRecord.ownerType)
		return
	}
	if !referenceInSet(reachable.index.declarations, reference) {
		return
	}
	reachable.declarations[reference] = true
	index, present := storedRecordIndex(
		reachable.pkg.declarations.records,
		reference,
		func(record storedDeclaration) declarationRef {
			return record.id
		},
	)
	if present {
		reachable.markType(
			reachable.pkg.declarations.records[index].typeID,
		)
	}
}

func (reachable *normalizedPackageReachability) markBinding(
	reference bindingRef,
) {
	if reference == 0 ||
		uint64(reference) >= uint64(len(reachable.bindings)) ||
		reachable.bindings[reference] ||
		!referenceInSet(reachable.index.bindings, reference) {
		return
	}
	reachable.bindings[reference] = true
	index, present := storedRecordIndex(
		reachable.pkg.bindings.records,
		reference,
		func(record storedBinding) bindingRef {
			return record.id
		},
	)
	if present {
		reachable.markType(
			reachable.pkg.bindings.records[index].typeID,
		)
	}
}

func (reachable *normalizedPackageReachability) markType(
	reference typeRef,
) {
	if reference == 0 ||
		uint64(reference) >= uint64(len(reachable.types)) ||
		reachable.types[reference] ||
		!referenceInSet(reachable.index.types, reference) {
		return
	}
	reachable.types[reference] = true
	reachable.typeQueue = append(reachable.typeQueue, reference)
}

func (reachable *normalizedPackageReachability) markOperation(
	reference operationRef,
) {
	if reference == 0 ||
		uint64(reference) >= uint64(len(reachable.operations)) ||
		reachable.operations[reference] ||
		!referenceInSet(reachable.index.operations, reference) {
		return
	}
	reachable.operations[reference] = true
	reachable.operationQueue = append(
		reachable.operationQueue, reference,
	)
}

func (reachable *normalizedPackageReachability) markUnsupported(
	reference unsupportedRef,
) {
	if reference != 0 &&
		uint64(reference) < uint64(len(reachable.unsupported)) &&
		referenceInSet(reachable.index.unsupported, reference) {
		reachable.unsupported[reference] = true
	}
}

func (reachable *normalizedPackageReachability) drainOperations() {
	for cursor := 0; cursor < len(reachable.operationQueue); cursor++ {
		reference := reachable.operationQueue[cursor]
		index, present := storedRecordIndex(
			reachable.pkg.operations.records,
			reference,
			func(record storedOperation) operationRef {
				return record.id
			},
		)
		if !present {
			continue
		}
		reachable.markOperationRecord(
			reachable.pkg.operations.records[index],
		)
	}
}

func (reachable *normalizedPackageReachability) markOperationRecord(
	record storedOperation,
) {
	store := reachable.pkg.operations
	reachable.markType(record.resultType)
	reachable.markType(record.expectedType)
	reachable.markObject(record.object)
	if record.selection != 0 {
		selection := store.selections[record.selection-1]
		reachable.markType(selection.receiver)
		reachable.markDeclaration(selection.object)
	}
	if record.instance != 0 {
		instance := store.instances[record.instance-1]
		reachable.markObject(instance.target)
		types, _ := storedRelation(
			store.instanceTypes,
			instance.types.start,
			instance.types.count,
		)
		for _, typeID := range types {
			reachable.markType(typeID)
		}
		reachable.markType(instance.signature)
	}
	implicit, _ := storedRelation(
		store.implicit,
		record.implicit.start,
		record.implicit.count,
	)
	for _, operation := range implicit {
		reachable.markType(operation.source)
		reachable.markType(operation.target)
	}
	reachable.markBinding(record.label)
	reachable.markOperation(record.controlTarget)
}

func (reachable *normalizedPackageReachability) markObject(
	reference objectReferenceRef,
) {
	if reference == 0 {
		return
	}
	object := reachable.pkg.operations.objects[reference-1]
	switch object.kind {
	case ObjectReferenceDeclaration:
		reachable.markDeclaration(object.declaration)
	case ObjectReferenceBinding:
		reachable.markBinding(object.binding)
	}
}

func (reachable *normalizedPackageReachability) drainTypes() {
	for cursor := 0; cursor < len(reachable.typeQueue); cursor++ {
		reference := reachable.typeQueue[cursor]
		index, present := storedRecordIndex(
			reachable.pkg.types.records,
			reference,
			func(record storedType) typeRef {
				return record.id
			},
		)
		if !present {
			continue
		}
		reachable.markTypeRecord(
			reachable.pkg.types.records[index],
		)
	}
}

func (reachable *normalizedPackageReachability) markTypeRecord(
	record storedType,
) {
	store := reachable.pkg.types
	switch record.kind {
	case TypeNamed, TypeAlias:
		payload := store.nominal[record.payload-1]
		reachable.markDeclaration(payload.declaration)
		reachable.markTypeRange(payload.arguments)
		reachable.markType(payload.target)
		reachable.markMethodTypes(payload.methods)
	case TypeParameter:
		payload := store.parameters[record.payload-1]
		reachable.markDeclaration(payload.declaration)
		reachable.markType(payload.constraint)
	case TypePointer, TypeSlice:
		reachable.markType(store.elements[record.payload-1])
	case TypeArray:
		reachable.markType(store.arrays[record.payload-1].element)
	case TypeMap:
		payload := store.maps[record.payload-1]
		reachable.markType(payload.key)
		reachable.markType(payload.element)
	case TypeChannel:
		reachable.markType(store.channels[record.payload-1].element)
	case TypeSignature:
		payload := store.signatures[record.payload-1]
		reachable.markType(payload.receiver)
		reachable.markTypeRange(payload.receiverTypeParameters)
		reachable.markTypeRange(payload.typeParameters)
		reachable.markTypeRange(payload.parameters)
		reachable.markTypeRange(payload.results)
	case TypeStruct:
		reachable.markFieldTypes(store.structs[record.payload-1])
	case TypeInterface:
		payload := store.interfaces[record.payload-1]
		reachable.markMethodTypes(payload.methods)
		reachable.markTypeRange(payload.embeddeds)
		reachable.markTermTypes(payload.terms)
	case TypeTuple:
		reachable.markTypeRange(store.tuples[record.payload-1])
	case TypeUnion:
		reachable.markTermTypes(store.unions[record.payload-1])
	}
}

func (reachable *normalizedPackageReachability) markTypeRange(
	relation typeRefRange,
) {
	types, _ := storedRelation(
		reachable.pkg.types.typeRelations,
		relation.start,
		relation.count,
	)
	for _, typeID := range types {
		reachable.markType(typeID)
	}
}

func (reachable *normalizedPackageReachability) markFieldTypes(
	relation typeFieldRange,
) {
	fields, _ := storedRelation(
		reachable.pkg.types.fields,
		relation.start,
		relation.count,
	)
	for _, field := range fields {
		reachable.markType(field.typeID)
	}
}

func (reachable *normalizedPackageReachability) markMethodTypes(
	relation typeMethodRange,
) {
	methods, _ := storedRelation(
		reachable.pkg.types.methods,
		relation.start,
		relation.count,
	)
	for _, method := range methods {
		reachable.markType(method.signature)
	}
}

func (reachable *normalizedPackageReachability) markTermTypes(
	relation typeTermRange,
) {
	terms, _ := storedRelation(
		reachable.pkg.types.terms,
		relation.start,
		relation.count,
	)
	for _, term := range terms {
		reachable.markType(term.typeID)
	}
}

func (reachable normalizedPackageReachability) requireComplete() error {
	if err := requireReachableRecords(
		"declaration",
		reachable.pkg.declarations.records,
		reachable.declarations,
		func(record storedDeclaration) declarationRef {
			return record.id
		},
		func(reference declarationRef) string {
			return reachable.pkg.identities.declaration(
				reference,
			).String()
		},
	); err != nil {
		return err
	}
	if err := requireReachableRecords(
		"binding",
		reachable.pkg.bindings.records,
		reachable.bindings,
		func(record storedBinding) bindingRef {
			return record.id
		},
		func(reference bindingRef) string {
			return reachable.pkg.identities.binding(
				reference,
			).String()
		},
	); err != nil {
		return err
	}
	if err := requireReachableRecords(
		"type",
		reachable.pkg.types.records,
		reachable.types,
		func(record storedType) typeRef {
			return record.id
		},
		func(reference typeRef) string {
			return reachable.pkg.identities.typeID(
				reference,
			).String()
		},
	); err != nil {
		return err
	}
	if err := requireReachableRecords(
		"operation",
		reachable.pkg.operations.records,
		reachable.operations,
		func(record storedOperation) operationRef {
			return record.id
		},
		func(reference operationRef) string {
			return reachable.pkg.identities.operation(
				reference,
			).String()
		},
	); err != nil {
		return err
	}
	return requireReachableRecords(
		"unsupported",
		reachable.pkg.unsupported.records,
		reachable.unsupported,
		func(record storedUnsupported) unsupportedRef {
			return record.id
		},
		func(reference unsupportedRef) string {
			return reachable.pkg.identities.unsupportedID(
				reference,
			).String()
		},
	)
}

func requireReachableRecords[
	Reference ~uint64,
	Record any,
](
	name string,
	records []Record,
	reached []bool,
	reference func(Record) Reference,
	render func(Reference) string,
) error {
	for _, record := range records {
		current := reference(record)
		if uint64(current) >= uint64(len(reached)) ||
			!reached[current] {
			return fmt.Errorf(
				"semantic package contains unreachable %s %s",
				name,
				render(current),
			)
		}
	}
	return nil
}
