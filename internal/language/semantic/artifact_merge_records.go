package semantic

import "github.com/tsoniclang/gotots/internal/identity"

func (merge *mixedShardMerge) definitions(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireDefinitionDecoder{
		identities: merge.checker.identities,
		authority:  merge.checkerAuthority,
	}
	providerDecoder := wireDefinitionDecoder{
		identities: merge.provider.identities,
		authority:  merge.providerAuthority,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"definitions",
		checkerCount,
		func(
			encoded wireDefinitionRecord,
			_ Authority,
		) (DefinitionSemantics, error) {
			return checkerDecoder.record(encoded)
		},
		func(record DefinitionSemantics) identity.DefinitionID {
			return record.Definition()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"definitions",
		providerCount,
		func(
			encoded wireDefinitionRecord,
			_ Authority,
		) (DefinitionSemantics, error) {
			return providerDecoder.record(encoded)
		},
		func(record DefinitionSemantics) identity.DefinitionID {
			return record.Definition()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalDefinitionRecords,
		func(record DefinitionSemantics) bool {
			return merge.projection.definitionIsLocal(
				record.Definition(),
			)
		},
		func(record DefinitionSemantics, _ Authority) error {
			merge.normalized.addDefinition(record)
			return nil
		},
	)
}

func (merge *mixedShardMerge) resolutions(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireResolutionDecoder{
		identities: merge.checker.identities,
	}
	providerDecoder := wireResolutionDecoder{
		identities: merge.provider.identities,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"resolutions",
		checkerCount,
		func(
			encoded wireResolutionRecord,
			_ Authority,
		) (OccurrenceResolution, error) {
			return checkerDecoder.record(encoded)
		},
		func(record OccurrenceResolution) identity.OccurrenceID {
			return record.Occurrence()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"resolutions",
		providerCount,
		func(
			encoded wireResolutionRecord,
			_ Authority,
		) (OccurrenceResolution, error) {
			return providerDecoder.record(encoded)
		},
		func(record OccurrenceResolution) identity.OccurrenceID {
			return record.Occurrence()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalResolutionRecords,
		func(record OccurrenceResolution) bool {
			file := record.Occurrence().Span().File()
			return merge.projection.localFiles[file]
		},
		func(record OccurrenceResolution, _ Authority) error {
			merge.normalized.addResolution(record)
			return nil
		},
	)
}

func (merge *mixedShardMerge) declarations(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireObjectDecoder{
		identities: merge.checker.identities,
		authority:  merge.checkerAuthority,
	}
	providerDecoder := wireObjectDecoder{
		identities: merge.provider.identities,
		authority:  merge.providerAuthority,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"declarations",
		checkerCount,
		func(
			encoded wireDeclarationRecord,
			_ Authority,
		) (Declaration, error) {
			return checkerDecoder.declaration(encoded)
		},
		func(record Declaration) identity.SemanticDeclarationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"declarations",
		providerCount,
		func(
			encoded wireDeclarationRecord,
			_ Authority,
		) (Declaration, error) {
			return providerDecoder.declaration(encoded)
		},
		func(record Declaration) identity.SemanticDeclarationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalDeclarationRecords,
		nil,
		func(record Declaration, _ Authority) error {
			merge.normalized.addDeclaration(record)
			return nil
		},
	)
}

func (merge *mixedShardMerge) bindings(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireObjectDecoder{
		identities: merge.checker.identities,
		authority:  merge.checkerAuthority,
	}
	providerDecoder := wireObjectDecoder{
		identities: merge.provider.identities,
		authority:  merge.providerAuthority,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"bindings",
		checkerCount,
		func(
			encoded wireBindingRecord,
			_ Authority,
		) (Binding, error) {
			return checkerDecoder.binding(encoded)
		},
		func(record Binding) identity.SemanticBindingID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"bindings",
		providerCount,
		func(
			encoded wireBindingRecord,
			_ Authority,
		) (Binding, error) {
			return providerDecoder.binding(encoded)
		},
		func(record Binding) identity.SemanticBindingID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalBindingRecords,
		nil,
		func(record Binding, _ Authority) error {
			merge.normalized.addBinding(record)
			return nil
		},
	)
}

func (merge *mixedShardMerge) types(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireTypeDecoder{
		identities: merge.checker.identities,
	}
	providerDecoder := wireTypeDecoder{
		identities: merge.provider.identities,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"types",
		checkerCount,
		func(encoded wireTypeRecord, _ Authority) (Type, error) {
			return checkerDecoder.record(encoded)
		},
		func(record Type) identity.SemanticTypeID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"types",
		providerCount,
		func(encoded wireTypeRecord, _ Authority) (Type, error) {
			return providerDecoder.record(encoded)
		},
		func(record Type) identity.SemanticTypeID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalTypeRecords,
		nil,
		func(record Type, authority Authority) error {
			witness, witnessErr := NewTypeWitness(
				merge.projection.id,
				record.ID(),
				authority,
			)
			if witnessErr != nil {
				return witnessErr
			}
			merge.normalized.addType(record)
			merge.normalized.addTypeWitness(witness)
			return nil
		},
	)
}

func (merge *mixedShardMerge) operations(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireOperationDecoder{
		identities: merge.checker.identities,
	}
	providerDecoder := wireOperationDecoder{
		identities: merge.provider.identities,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"operations",
		checkerCount,
		func(encoded wireOperationRecord, _ Authority) (Operation, error) {
			return checkerDecoder.record(encoded)
		},
		func(record Operation) identity.OperationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"operations",
		providerCount,
		func(encoded wireOperationRecord, _ Authority) (Operation, error) {
			return providerDecoder.record(encoded)
		},
		func(record Operation) identity.OperationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalOperationRecords,
		nil,
		func(record Operation, _ Authority) error {
			merge.normalized.addOperation(record)
			return nil
		},
	)
}

func (merge *mixedShardMerge) unsupported(
	checkerCount int,
	providerCount int,
) error {
	checkerDecoder := wireObjectDecoder{
		identities: merge.checker.identities,
		authority:  merge.checkerAuthority,
	}
	providerDecoder := wireObjectDecoder{
		identities: merge.provider.identities,
		authority:  merge.providerAuthority,
	}
	checker, err := openDecodedRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"unsupported",
		checkerCount,
		func(
			encoded wireUnsupportedRecord,
			_ Authority,
		) (Unsupported, error) {
			return checkerDecoder.unsupported(encoded)
		},
		func(record Unsupported) identity.UnsupportedID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openDecodedRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"unsupported",
		providerCount,
		func(
			encoded wireUnsupportedRecord,
			_ Authority,
		) (Unsupported, error) {
			return providerDecoder.unsupported(encoded)
		},
		func(record Unsupported) identity.UnsupportedID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeDecodedRecords(
		checker,
		provider,
		equalUnsupportedRecords,
		nil,
		func(record Unsupported, _ Authority) error {
			merge.normalized.addUnsupported(record)
			return nil
		},
	)
}
