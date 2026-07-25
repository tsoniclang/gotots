package semantic

import "fmt"

func validateTypeStorage(
	store packageTypeStore,
) error {
	payloads := typePayloadArenas(store)
	relations := typeRelationArenas(store)
	for _, record := range store.records {
		arena := payloads[record.kind]
		if arena == nil {
			return fmt.Errorf(
				"semantic type storage has invalid kind %d",
				record.kind,
			)
		}
		if err := arena.payload(record.payload); err != nil {
			return err
		}
	}
	for _, value := range store.nominal {
		if err := relations.types.relation(
			value.arguments.start, value.arguments.count,
		); err != nil {
			return err
		}
		if err := relations.methods.relation(
			value.methods.start, value.methods.count,
		); err != nil {
			return err
		}
	}
	for _, value := range store.signatures {
		ranges := [...]typeRefRange{
			value.receiverTypeParameters,
			value.typeParameters,
			value.parameters,
			value.results,
		}
		for _, current := range ranges {
			if err := relations.types.relation(
				current.start, current.count,
			); err != nil {
				return err
			}
		}
	}
	for _, value := range store.structs {
		if err := relations.fields.relation(
			value.start, value.count,
		); err != nil {
			return err
		}
	}
	for _, value := range store.interfaces {
		if err := relations.methods.relation(
			value.methods.start, value.methods.count,
		); err != nil {
			return err
		}
		if err := relations.types.relation(
			value.embeddeds.start, value.embeddeds.count,
		); err != nil {
			return err
		}
		if err := relations.terms.relation(
			value.terms.start, value.terms.count,
		); err != nil {
			return err
		}
	}
	for _, value := range store.tuples {
		if err := relations.types.relation(
			value.start, value.count,
		); err != nil {
			return err
		}
	}
	for _, value := range store.unions {
		if err := relations.terms.relation(
			value.start, value.count,
		); err != nil {
			return err
		}
	}
	for _, arena := range payloads {
		if err := arena.complete(); err != nil {
			return err
		}
	}
	return completeStorageArenas(
		relations.types,
		relations.fields,
		relations.methods,
		relations.terms,
	)
}

func typePayloadArenas(
	store packageTypeStore,
) map[TypeKind]*storageArenaUse {
	arenas := map[TypeKind]*storageArenaUse{}
	add := func(kinds []TypeKind, name string, size int) {
		value := newStorageArenaUse(name, size)
		for _, kind := range kinds {
			arenas[kind] = &value
		}
	}
	add([]TypeKind{TypeBasic}, "basic-type", len(store.basic))
	add(
		[]TypeKind{TypeNamed, TypeAlias},
		"nominal-type",
		len(store.nominal),
	)
	add(
		[]TypeKind{TypeParameter},
		"type-parameter",
		len(store.parameters),
	)
	add(
		[]TypeKind{TypePointer, TypeSlice},
		"element-type",
		len(store.elements),
	)
	add([]TypeKind{TypeArray}, "array-type", len(store.arrays))
	add([]TypeKind{TypeMap}, "map-type", len(store.maps))
	add([]TypeKind{TypeChannel}, "channel-type", len(store.channels))
	add(
		[]TypeKind{TypeSignature},
		"signature-type",
		len(store.signatures),
	)
	add([]TypeKind{TypeStruct}, "struct-type", len(store.structs))
	add(
		[]TypeKind{TypeInterface},
		"interface-type",
		len(store.interfaces),
	)
	add([]TypeKind{TypeTuple}, "tuple-type", len(store.tuples))
	add([]TypeKind{TypeUnion}, "union-type", len(store.unions))
	return arenas
}

type typeStorageRelations struct {
	types   storageArenaUse
	fields  storageArenaUse
	methods storageArenaUse
	terms   storageArenaUse
}

func typeRelationArenas(
	store packageTypeStore,
) typeStorageRelations {
	return typeStorageRelations{
		types: newStorageArenaUse(
			"type-reference", len(store.typeRelations),
		),
		fields: newStorageArenaUse(
			"type-field", len(store.fields),
		),
		methods: newStorageArenaUse(
			"type-method", len(store.methods),
		),
		terms: newStorageArenaUse(
			"type-term", len(store.terms),
		),
	}
}
