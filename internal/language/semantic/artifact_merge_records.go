package semantic

import "github.com/tsoniclang/gotots/internal/identity"

func (merge *mixedBinaryShardMerge) definitions(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"definitions",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (DefinitionSemantics, error) {
			return decodeBinaryDefinitionValue(
				decoder, merge.checker.identities.table, authority,
			)
		},
		func(record DefinitionSemantics) identity.DefinitionID {
			return record.Definition()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"definitions",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (DefinitionSemantics, error) {
			return decodeBinaryDefinitionValue(
				decoder, merge.provider.identities.table, authority,
			)
		},
		func(record DefinitionSemantics) identity.DefinitionID {
			return record.Definition()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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

func (merge *mixedBinaryShardMerge) resolutions(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"resolutions",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			_ Authority,
		) (OccurrenceResolution, error) {
			return decodeBinaryResolutionValue(
				decoder, merge.checker.identities.table,
			)
		},
		func(record OccurrenceResolution) identity.OccurrenceID {
			return record.Occurrence()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"resolutions",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			_ Authority,
		) (OccurrenceResolution, error) {
			return decodeBinaryResolutionValue(
				decoder, merge.provider.identities.table,
			)
		},
		func(record OccurrenceResolution) identity.OccurrenceID {
			return record.Occurrence()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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

func (merge *mixedBinaryShardMerge) declarations(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"declarations",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (Declaration, error) {
			return decodeBinaryDeclarationValue(
				decoder, merge.checker.identities.table, authority,
			)
		},
		func(record Declaration) identity.SemanticDeclarationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"declarations",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (Declaration, error) {
			return decodeBinaryDeclarationValue(
				decoder, merge.provider.identities.table, authority,
			)
		},
		func(record Declaration) identity.SemanticDeclarationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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

func (merge *mixedBinaryShardMerge) bindings(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"bindings",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (Binding, error) {
			return decodeBinaryBindingValue(
				decoder, merge.checker.identities.table, authority,
			)
		},
		func(record Binding) identity.SemanticBindingID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"bindings",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (Binding, error) {
			return decodeBinaryBindingValue(
				decoder, merge.provider.identities.table, authority,
			)
		},
		func(record Binding) identity.SemanticBindingID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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

func (merge *mixedBinaryShardMerge) types(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"types",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			_ Authority,
		) (Type, error) {
			return decodeBinaryTypeValue(
				decoder, merge.checker.identities.table,
			)
		},
		func(record Type) identity.SemanticTypeID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"types",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			_ Authority,
		) (Type, error) {
			return decodeBinaryTypeValue(
				decoder, merge.provider.identities.table,
			)
		},
		func(record Type) identity.SemanticTypeID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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

func (merge *mixedBinaryShardMerge) operations(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"operations",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			_ Authority,
		) (Operation, error) {
			return decodeBinaryOperationValue(
				decoder, merge.checker.identities.table,
			)
		},
		func(record Operation) identity.OperationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"operations",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			_ Authority,
		) (Operation, error) {
			return decodeBinaryOperationValue(
				decoder, merge.provider.identities.table,
			)
		},
		func(record Operation) identity.OperationID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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

func (merge *mixedBinaryShardMerge) unsupported(
	checkerCount int,
	providerCount int,
) error {
	checker, err := openBinaryRecordCursor(
		merge.checker.decoder,
		merge.checkerAuthority,
		"unsupported",
		checkerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (Unsupported, error) {
			return decodeBinaryUnsupportedValue(
				decoder, merge.checker.identities.table, authority,
			)
		},
		func(record Unsupported) identity.UnsupportedID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	provider, err := openBinaryRecordCursor(
		merge.provider.decoder,
		merge.providerAuthority,
		"unsupported",
		providerCount,
		func(
			decoder *binaryShardDecoder,
			authority Authority,
		) (Unsupported, error) {
			return decodeBinaryUnsupportedValue(
				decoder, merge.provider.identities.table, authority,
			)
		},
		func(record Unsupported) identity.UnsupportedID {
			return record.ID()
		},
	)
	if err != nil {
		return err
	}
	return mergeBinaryRecords(
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
