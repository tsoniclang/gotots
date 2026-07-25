package semantic

import "fmt"

func validateDefinitionStorage(
	store packageDefinitionStore,
) error {
	callable := newStorageArenaUse(
		"callable-definition", len(store.callable),
	)
	initializers := newStorageArenaUse(
		"initializer-definition", len(store.initializers),
	)
	bodyless := newStorageArenaUse(
		"bodyless-definition", len(store.bodyless),
	)
	implicit := newStorageArenaUse(
		"implicit-definition", len(store.implicit),
	)
	synthetic := newStorageArenaUse(
		"synthetic-definition", len(store.synthetic),
	)
	declarations := newStorageArenaUse(
		"definition-declaration", len(store.declarationRelations),
	)
	bindings := newStorageArenaUse(
		"definition-binding", len(store.bindingRelations),
	)
	entries := newStorageArenaUse(
		"initializer-entry", len(store.initializerEntries),
	)
	for _, record := range store.records {
		if err := bindings.relation(
			record.bindings.start, record.bindings.count,
		); err != nil {
			return err
		}
		switch record.form {
		case DefinitionFormCallable:
			if err := callable.payload(record.payload); err != nil {
				return err
			}
			value := store.callable[record.payload-1]
			if err := declarations.relation(
				value.declarations.start,
				value.declarations.count,
			); err != nil {
				return err
			}
		case DefinitionFormInitializer:
			if err := initializers.payload(record.payload); err != nil {
				return err
			}
			value := store.initializers[record.payload-1]
			if err := declarations.relation(
				value.declarations.start,
				value.declarations.count,
			); err != nil {
				return err
			}
			if err := entries.relation(
				value.entries.start, value.entries.count,
			); err != nil {
				return err
			}
		case DefinitionFormBodyless:
			if err := bodyless.payload(record.payload); err != nil {
				return err
			}
		case DefinitionFormImplicit:
			if err := implicit.payload(record.payload); err != nil {
				return err
			}
		case DefinitionFormSynthetic:
			if err := synthetic.payload(record.payload); err != nil {
				return err
			}
		default:
			return fmt.Errorf(
				"semantic definition storage has invalid form %d",
				record.form,
			)
		}
	}
	return completeStorageArenas(
		callable,
		initializers,
		bodyless,
		implicit,
		synthetic,
		declarations,
		bindings,
		entries,
	)
}

func validateResolutionStorage(
	store packageResolutionStore,
) error {
	arenas := map[ResolutionKind]*storageArenaUse{}
	newArena := func(
		kind ResolutionKind,
		name string,
		size int,
	) {
		value := newStorageArenaUse(name, size)
		arenas[kind] = &value
	}
	newArena(
		ResolutionStructuralOnly,
		"structural-resolution",
		len(store.structural),
	)
	newArena(
		ResolutionDefinitionComponent,
		"definition-component-resolution",
		len(store.definitionComponents),
	)
	newArena(
		ResolutionDeclaration,
		"declaration-resolution",
		len(store.declarations),
	)
	newArena(
		ResolutionBinding,
		"binding-resolution",
		len(store.bindings),
	)
	newArena(ResolutionType, "type-resolution", len(store.types))
	newArena(
		ResolutionOperation,
		"operation-resolution",
		len(store.operations),
	)
	newArena(
		ResolutionUnsupported,
		"unsupported-resolution",
		len(store.unsupported),
	)
	for _, record := range store.records {
		arena := arenas[record.kind]
		if arena == nil {
			return fmt.Errorf(
				"semantic resolution storage has invalid kind %d",
				record.kind,
			)
		}
		if err := arena.payload(record.payload); err != nil {
			return err
		}
	}
	for _, arena := range arenas {
		if err := arena.complete(); err != nil {
			return err
		}
	}
	return nil
}

func validateBindingStorage(
	store packageBindingStore,
) error {
	captures := newStorageArenaUse(
		"binding-capture", len(store.captures),
	)
	for _, record := range store.records {
		if err := captures.relation(
			record.captures.start,
			record.captures.count,
		); err != nil {
			return err
		}
	}
	return captures.complete()
}
