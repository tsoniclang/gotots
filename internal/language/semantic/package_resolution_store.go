package semantic

import (
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/language/catalog"
)

type storedResolution struct {
	occurrence occurrenceRef
	owner      definitionRef
	syntax     catalog.Kind
	role       catalog.Role
	variant    catalog.Variant
	domain     catalog.ResolutionDomain
	kind       ResolutionKind
	payload    uint64
}

type storedStructuralResolution struct {
	disposition StructuralDisposition
	declaration declarationRef
	typeID      typeRef
}

type storedDefinitionComponent struct {
	component  DefinitionComponentKind
	definition definitionRef
}

type packageResolutionBuilder struct {
	records              []storedResolution
	structural           []storedStructuralResolution
	definitionComponents []storedDefinitionComponent
	declarations         []declarationRef
	bindings             []bindingRef
	types                []typeRef
	operations           []operationRef
	unsupported          []unsupportedRef
}

func (builder *packageResolutionBuilder) add(
	identities *packageIdentityBuilder,
	record OccurrenceResolution,
) {
	stored := storedResolution{
		occurrence: identities.occurrence(record.occurrence),
		owner:      identities.definition(record.owner),
		syntax:     record.syntax,
		role:       record.role,
		variant:    record.variant,
		domain:     record.domain,
		kind:       record.kind,
	}
	switch record.kind {
	case ResolutionStructuralOnly:
		builder.structural = append(
			builder.structural,
			storedStructuralResolution{
				disposition: record.structural.disposition,
				declaration: identities.declaration(
					record.structural.declaration,
				),
				typeID: identities.typeID(record.structural.typeID),
			},
		)
		stored.payload = uint64(len(builder.structural))
	case ResolutionDefinitionComponent:
		builder.definitionComponents = append(
			builder.definitionComponents,
			storedDefinitionComponent{
				component:  record.component,
				definition: identities.definition(record.definition),
			},
		)
		stored.payload = uint64(len(builder.definitionComponents))
	case ResolutionDeclaration:
		builder.declarations = append(
			builder.declarations,
			identities.declaration(record.declaration),
		)
		stored.payload = uint64(len(builder.declarations))
	case ResolutionBinding:
		builder.bindings = append(
			builder.bindings,
			identities.binding(record.binding),
		)
		stored.payload = uint64(len(builder.bindings))
	case ResolutionType:
		builder.types = append(
			builder.types,
			identities.typeID(record.typeID),
		)
		stored.payload = uint64(len(builder.types))
	case ResolutionOperation:
		builder.operations = append(
			builder.operations,
			identities.operation(record.operation),
		)
		stored.payload = uint64(len(builder.operations))
	case ResolutionUnsupported:
		builder.unsupported = append(
			builder.unsupported,
			identities.unsupportedID(record.unsupported),
		)
		stored.payload = uint64(len(builder.unsupported))
	}
	builder.records = append(builder.records, stored)
}

type packageResolutionStore struct {
	records              []storedResolution
	structural           []storedStructuralResolution
	definitionComponents []storedDefinitionComponent
	declarations         []declarationRef
	bindings             []bindingRef
	types                []typeRef
	operations           []operationRef
	unsupported          []unsupportedRef
}

func (builder *packageResolutionBuilder) seal(
	remap packageIdentityRemap,
) (packageResolutionStore, error) {
	store := packageResolutionStore{
		records:              builder.records,
		structural:           builder.structural,
		definitionComponents: builder.definitionComponents,
		declarations:         builder.declarations,
		bindings:             builder.bindings,
		types:                builder.types,
		operations:           builder.operations,
		unsupported:          builder.unsupported,
	}
	var err error
	for index := range store.records {
		record := &store.records[index]
		record.occurrence, err = remapReference(
			record.occurrence, remap.occurrences,
		)
		if err != nil {
			return packageResolutionStore{}, err
		}
		record.owner, err = remapReference(
			record.owner, remap.definitions,
		)
		if err != nil {
			return packageResolutionStore{}, err
		}
	}
	for index := range store.structural {
		payload := &store.structural[index]
		payload.declaration, err = remapReference(
			payload.declaration, remap.declarations,
		)
		if err != nil {
			return packageResolutionStore{}, err
		}
		payload.typeID, err = remapReference(
			payload.typeID, remap.types,
		)
		if err != nil {
			return packageResolutionStore{}, err
		}
	}
	for index := range store.definitionComponents {
		payload := &store.definitionComponents[index]
		payload.definition, err = remapReference(
			payload.definition, remap.definitions,
		)
		if err != nil {
			return packageResolutionStore{}, err
		}
	}
	if err := remapReferences(store.declarations, remap.declarations); err != nil {
		return packageResolutionStore{}, err
	}
	if err := remapReferences(store.bindings, remap.bindings); err != nil {
		return packageResolutionStore{}, err
	}
	if err := remapReferences(store.types, remap.types); err != nil {
		return packageResolutionStore{}, err
	}
	if err := remapReferences(store.operations, remap.operations); err != nil {
		return packageResolutionStore{}, err
	}
	if err := remapReferences(store.unsupported, remap.unsupported); err != nil {
		return packageResolutionStore{}, err
	}
	sort.Slice(store.records, func(left, right int) bool {
		return store.records[left].occurrence <
			store.records[right].occurrence
	})
	return store, nil
}

func remapReferences[Ref ~uint64](
	references []Ref,
	remap []uint64,
) error {
	for index := range references {
		value, err := remapReference(references[index], remap)
		if err != nil {
			return err
		}
		references[index] = value
	}
	return nil
}

func payloadAt[Value any](
	values []Value,
	reference uint64,
) (Value, error) {
	if reference == 0 || reference > uint64(len(values)) {
		var zero Value
		return zero, fmt.Errorf(
			"semantic package payload reference %d is invalid",
			reference,
		)
	}
	return values[reference-1], nil
}

func (store packageResolutionStore) record(
	identities *packageIdentityProjection,
	index int,
) (OccurrenceResolution, error) {
	if index < 0 || index >= len(store.records) {
		return OccurrenceResolution{}, fmt.Errorf(
			"semantic resolution index %d is invalid", index,
		)
	}
	stored := store.records[index]
	record := OccurrenceResolution{
		occurrence: identities.occurrence(stored.occurrence),
		owner:      identities.definition(stored.owner),
		syntax:     stored.syntax,
		role:       stored.role,
		variant:    stored.variant,
		domain:     stored.domain,
		kind:       stored.kind,
	}
	switch stored.kind {
	case ResolutionStructuralOnly:
		payload, err := payloadAt(store.structural, stored.payload)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.structural = StructuralEvidence{
			disposition: payload.disposition,
			declaration: identities.declaration(payload.declaration),
			typeID:      identities.typeID(payload.typeID),
		}
	case ResolutionDefinitionComponent:
		payload, err := payloadAt(
			store.definitionComponents, stored.payload,
		)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.component = payload.component
		record.definition = identities.definition(payload.definition)
	case ResolutionDeclaration:
		payload, err := payloadAt(store.declarations, stored.payload)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.declaration = identities.declaration(payload)
	case ResolutionBinding:
		payload, err := payloadAt(store.bindings, stored.payload)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.binding = identities.binding(payload)
	case ResolutionType:
		payload, err := payloadAt(store.types, stored.payload)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.typeID = identities.typeID(payload)
	case ResolutionOperation:
		payload, err := payloadAt(store.operations, stored.payload)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.operation = identities.operation(payload)
	case ResolutionUnsupported:
		payload, err := payloadAt(store.unsupported, stored.payload)
		if err != nil {
			return OccurrenceResolution{}, err
		}
		record.unsupported = identities.unsupportedID(payload)
	default:
		return OccurrenceResolution{}, fmt.Errorf(
			"semantic resolution has invalid kind %d", stored.kind,
		)
	}
	return record, nil
}
