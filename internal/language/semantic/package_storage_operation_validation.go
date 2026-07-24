package semantic

func validateOperationStorage(
	store packageOperationStore,
) error {
	constants := newStorageArenaUse(
		"operation-constant", len(store.constants),
	)
	objects := newStorageArenaUse(
		"operation-object", len(store.objects),
	)
	selections := newStorageArenaUse(
		"operation-selection", len(store.selections),
	)
	instances := newStorageArenaUse(
		"operation-instance", len(store.instances),
	)
	operands := newStorageArenaUse(
		"operation-operand", len(store.operands),
	)
	definitions := newStorageArenaUse(
		"operation-definition", len(store.definitions),
	)
	implicit := newStorageArenaUse(
		"implicit-operation", len(store.implicit),
	)
	indexes := newStorageArenaUse(
		"selection-index", len(store.selectionIndexes),
	)
	instanceTypes := newStorageArenaUse(
		"instance-type", len(store.instanceTypes),
	)
	for _, record := range store.records {
		if err := optionalStoragePayload(
			&constants, uint64(record.constant),
		); err != nil {
			return err
		}
		if err := optionalStoragePayload(
			&objects, uint64(record.object),
		); err != nil {
			return err
		}
		if err := optionalStoragePayload(
			&selections, uint64(record.selection),
		); err != nil {
			return err
		}
		if err := optionalStoragePayload(
			&instances, uint64(record.instance),
		); err != nil {
			return err
		}
		if err := operands.relation(
			record.operands.start, record.operands.count,
		); err != nil {
			return err
		}
		if err := definitions.relation(
			record.definitions.start, record.definitions.count,
		); err != nil {
			return err
		}
		if err := implicit.relation(
			record.implicit.start, record.implicit.count,
		); err != nil {
			return err
		}
	}
	for _, value := range store.selections {
		if err := indexes.relation(
			value.index.start, value.index.count,
		); err != nil {
			return err
		}
	}
	for _, value := range store.instances {
		if err := objects.payload(uint64(value.target)); err != nil {
			return err
		}
		if err := instanceTypes.relation(
			value.types.start, value.types.count,
		); err != nil {
			return err
		}
	}
	return completeStorageArenas(
		constants,
		objects,
		selections,
		instances,
		operands,
		definitions,
		implicit,
		indexes,
		instanceTypes,
	)
}

func optionalStoragePayload(
	arena *storageArenaUse,
	reference uint64,
) error {
	if reference == 0 {
		return nil
	}
	return arena.payload(reference)
}
