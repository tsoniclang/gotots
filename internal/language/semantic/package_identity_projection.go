package semantic

import "github.com/tsoniclang/gotots/internal/identity"

func componentAt[Ref ~uint64, Value any](
	values []Value,
	reference Ref,
) (Value, bool) {
	index := uint64(reference)
	if index == 0 || index > uint64(len(values)) {
		var zero Value
		return zero, false
	}
	return values[index-1], true
}

type identityProjectionValue[Reference comparable, Value any] struct {
	reference Reference
	value     Value
	present   bool
}

func (cached *identityProjectionValue[Reference, Value]) get(
	reference Reference,
	project func() Value,
) Value {
	if cached.present && cached.reference == reference {
		return cached.value
	}
	value := project()
	cached.reference = reference
	cached.value = value
	cached.present = true
	return value
}

// packageIdentityProjection reconstructs one streaming semantic value while
// retaining only its current component chain. It is constant-size and never
// becomes an expanded package identity table.
type packageIdentityProjection struct {
	table           packageIdentityTable
	moduleValue     identityProjectionValue[moduleRef, identity.ModuleID]
	ownerValue      identityProjectionValue[ownerRef, identity.Owner]
	packageValue    identityProjectionValue[packageRef, identity.PackageID]
	fileValue       identityProjectionValue[fileRef, identity.FileID]
	spanValue       identityProjectionValue[spanRef, identity.SpanID]
	occurrenceValue identityProjectionValue[
		occurrenceRef,
		identity.OccurrenceID,
	]
	definitionValue identityProjectionValue[
		definitionRef,
		identity.DefinitionID,
	]
	typeValue        identityProjectionValue[typeRef, identity.SemanticTypeID]
	declarationValue identityProjectionValue[
		declarationRef,
		identity.SemanticDeclarationID,
	]
	bindingValue identityProjectionValue[
		bindingRef,
		identity.SemanticBindingID,
	]
	operationValue identityProjectionValue[
		operationRef,
		identity.OperationID,
	]
	unsupportedValue identityProjectionValue[
		unsupportedRef,
		identity.UnsupportedID,
	]
}

func newPackageIdentityProjection(
	table packageIdentityTable,
) *packageIdentityProjection {
	return &packageIdentityProjection{table: table}
}

func (projection *packageIdentityProjection) projectModule(
	reference moduleRef,
) identity.ModuleID {
	return projection.moduleValue.get(reference, func() identity.ModuleID {
		record, present := componentAt(
			projection.table.modules, reference,
		)
		if !present {
			return identity.ModuleID{}
		}
		value, err := identity.NewModuleID(
			record.path, record.version,
		)
		if err != nil {
			return identity.ModuleID{}
		}
		return value
	})
}

func (projection *packageIdentityProjection) projectOwner(
	reference ownerRef,
) identity.Owner {
	return projection.ownerValue.get(reference, func() identity.Owner {
		record, present := componentAt(
			projection.table.owners, reference,
		)
		if !present {
			return identity.Owner{}
		}
		value, err := projectOwnerIdentity(projection, record)
		if err != nil {
			return identity.Owner{}
		}
		return value
	})
}

func (projection *packageIdentityProjection) projectPackage(
	reference packageRef,
) identity.PackageID {
	return projection.packageValue.get(
		reference,
		func() identity.PackageID {
			record, present := componentAt(
				projection.table.packages, reference,
			)
			if !present {
				return identity.PackageID{}
			}
			value, err := identity.NewPackageID(
				projection.projectOwner(record.owner),
				record.importPath,
			)
			if err != nil {
				return identity.PackageID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectFile(
	reference fileRef,
) identity.FileID {
	return projection.fileValue.get(reference, func() identity.FileID {
		record, present := componentAt(
			projection.table.files, reference,
		)
		if !present {
			return identity.FileID{}
		}
		value, err := identity.NewFileID(
			projection.projectOwner(record.owner),
			record.rel,
		)
		if err != nil {
			return identity.FileID{}
		}
		return value
	})
}

func (projection *packageIdentityProjection) projectSpan(
	reference spanRef,
) identity.SpanID {
	return projection.spanValue.get(reference, func() identity.SpanID {
		record, present := componentAt(
			projection.table.spans, reference,
		)
		if !present {
			return identity.SpanID{}
		}
		value, err := identity.NewSpanID(
			projection.projectFile(record.file),
			record.start,
			record.end,
		)
		if err != nil {
			return identity.SpanID{}
		}
		return value
	})
}

func (projection *packageIdentityProjection) projectOccurrence(
	reference occurrenceRef,
) identity.OccurrenceID {
	return projection.occurrenceValue.get(
		reference,
		func() identity.OccurrenceID {
			record, present := componentAt(
				projection.table.occurrences, reference,
			)
			if !present {
				return identity.OccurrenceID{}
			}
			value, err := identity.NewOccurrenceID(
				projection.projectSpan(record.span),
				record.kind,
			)
			if err != nil {
				return identity.OccurrenceID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectDefinition(
	reference definitionRef,
) identity.DefinitionID {
	return projection.definitionValue.get(
		reference,
		func() identity.DefinitionID {
			record, present := componentAt(
				projection.table.definitions, reference,
			)
			if !present {
				return identity.DefinitionID{}
			}
			value, err := projectDefinitionIdentity(
				projection, record,
			)
			if err != nil {
				return identity.DefinitionID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectType(
	reference typeRef,
) identity.SemanticTypeID {
	return projection.typeValue.get(
		reference,
		func() identity.SemanticTypeID {
			record, present := componentAt(
				projection.table.types, reference,
			)
			if !present {
				return identity.SemanticTypeID{}
			}
			value, err := identity.NewSemanticTypeID(record.digest)
			if err != nil {
				return identity.SemanticTypeID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectDeclaration(
	reference declarationRef,
) identity.SemanticDeclarationID {
	return projection.declarationValue.get(
		reference,
		func() identity.SemanticDeclarationID {
			record, present := componentAt(
				projection.table.declarations, reference,
			)
			if !present {
				return identity.SemanticDeclarationID{}
			}
			value, err := projectDeclarationIdentity(
				projection, record,
			)
			if err != nil {
				return identity.SemanticDeclarationID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectBinding(
	reference bindingRef,
) identity.SemanticBindingID {
	return projection.bindingValue.get(
		reference,
		func() identity.SemanticBindingID {
			record, present := componentAt(
				projection.table.bindings, reference,
			)
			if !present {
				return identity.SemanticBindingID{}
			}
			value, err := identity.NewSemanticBindingID(
				projection.projectOccurrence(record.owner),
				projection.projectOccurrence(record.declaration),
				record.role,
				record.ordinal,
			)
			if err != nil {
				return identity.SemanticBindingID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectOperation(
	reference operationRef,
) identity.OperationID {
	return projection.operationValue.get(
		reference,
		func() identity.OperationID {
			record, present := componentAt(
				projection.table.operations, reference,
			)
			if !present {
				return identity.OperationID{}
			}
			value, err := projectOperationIdentity(
				projection, record,
			)
			if err != nil {
				return identity.OperationID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) projectUnsupported(
	reference unsupportedRef,
) identity.UnsupportedID {
	return projection.unsupportedValue.get(
		reference,
		func() identity.UnsupportedID {
			record, present := componentAt(
				projection.table.unsupported, reference,
			)
			if !present {
				return identity.UnsupportedID{}
			}
			value, err := identity.NewUnsupportedID(
				projection.projectDefinition(record.definition),
				projection.projectOccurrence(record.occurrence),
			)
			if err != nil {
				return identity.UnsupportedID{}
			}
			return value
		},
	)
}

func (projection *packageIdentityProjection) module(
	reference moduleRef,
) identity.ModuleID {
	return projection.projectModule(reference)
}

func (projection *packageIdentityProjection) owner(
	reference ownerRef,
) identity.Owner {
	return projection.projectOwner(reference)
}

func (projection *packageIdentityProjection) packageID(
	reference packageRef,
) identity.PackageID {
	return projection.projectPackage(reference)
}

func (projection *packageIdentityProjection) file(
	reference fileRef,
) identity.FileID {
	return projection.projectFile(reference)
}

func (projection *packageIdentityProjection) span(
	reference spanRef,
) identity.SpanID {
	return projection.projectSpan(reference)
}

func (projection *packageIdentityProjection) occurrence(
	reference occurrenceRef,
) identity.OccurrenceID {
	return projection.projectOccurrence(reference)
}

func (projection *packageIdentityProjection) definition(
	reference definitionRef,
) identity.DefinitionID {
	return projection.projectDefinition(reference)
}

func (projection *packageIdentityProjection) typeID(
	reference typeRef,
) identity.SemanticTypeID {
	return projection.projectType(reference)
}

func (projection *packageIdentityProjection) declaration(
	reference declarationRef,
) identity.SemanticDeclarationID {
	return projection.projectDeclaration(reference)
}

func (projection *packageIdentityProjection) binding(
	reference bindingRef,
) identity.SemanticBindingID {
	return projection.projectBinding(reference)
}

func (projection *packageIdentityProjection) operation(
	reference operationRef,
) identity.OperationID {
	return projection.projectOperation(reference)
}

func (projection *packageIdentityProjection) unsupportedID(
	reference unsupportedRef,
) identity.UnsupportedID {
	return projection.projectUnsupported(reference)
}

func (table packageIdentityTable) module(
	reference moduleRef,
) identity.ModuleID {
	return newPackageIdentityProjection(table).projectModule(reference)
}

func (table packageIdentityTable) owner(
	reference ownerRef,
) identity.Owner {
	return newPackageIdentityProjection(table).projectOwner(reference)
}

func (table packageIdentityTable) packageID(
	reference packageRef,
) identity.PackageID {
	return newPackageIdentityProjection(table).projectPackage(reference)
}

func (table packageIdentityTable) file(
	reference fileRef,
) identity.FileID {
	return newPackageIdentityProjection(table).projectFile(reference)
}

func (table packageIdentityTable) span(
	reference spanRef,
) identity.SpanID {
	return newPackageIdentityProjection(table).projectSpan(reference)
}

func (table packageIdentityTable) occurrence(
	reference occurrenceRef,
) identity.OccurrenceID {
	return newPackageIdentityProjection(table).projectOccurrence(reference)
}

func (table packageIdentityTable) definition(
	reference definitionRef,
) identity.DefinitionID {
	return newPackageIdentityProjection(table).projectDefinition(reference)
}

func (table packageIdentityTable) typeID(
	reference typeRef,
) identity.SemanticTypeID {
	return newPackageIdentityProjection(table).projectType(reference)
}

func (table packageIdentityTable) declaration(
	reference declarationRef,
) identity.SemanticDeclarationID {
	return newPackageIdentityProjection(table).projectDeclaration(reference)
}

func (table packageIdentityTable) binding(
	reference bindingRef,
) identity.SemanticBindingID {
	return newPackageIdentityProjection(table).projectBinding(reference)
}

func (table packageIdentityTable) operation(
	reference operationRef,
) identity.OperationID {
	return newPackageIdentityProjection(table).projectOperation(reference)
}

func (table packageIdentityTable) unsupportedID(
	reference unsupportedRef,
) identity.UnsupportedID {
	return newPackageIdentityProjection(table).projectUnsupported(reference)
}
