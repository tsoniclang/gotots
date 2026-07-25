package semantic

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
)

func storedIdentityReference[Record any, Reference ~uint64](
	records []Record,
	target Record,
	less func(Record, Record) bool,
) Reference {
	index := sort.Search(len(records), func(index int) bool {
		return !less(records[index], target)
	})
	if index == len(records) ||
		less(target, records[index]) ||
		less(records[index], target) {
		return 0
	}
	return Reference(index + 1)
}

func (table packageIdentityTable) moduleReference(
	value identity.ModuleID,
) moduleRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedModuleIdentity,
		moduleRef,
	](
		table.modules,
		storedModuleIdentity{
			path: value.Path(), version: value.Version(),
		},
		lessStoredModuleIdentity,
	)
}

func (table packageIdentityTable) ownerReference(
	value identity.Owner,
) ownerRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedOwnerIdentity,
		ownerRef,
	](
		table.owners,
		storedOwnerIdentity{
			class:  value.Class(),
			module: table.moduleReference(value.Module()),
		},
		lessStoredOwnerIdentity,
	)
}

func (table packageIdentityTable) packageReference(
	value identity.PackageID,
) packageRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedPackageIdentity,
		packageRef,
	](
		table.packages,
		storedPackageIdentity{
			owner:      table.ownerReference(value.Owner()),
			importPath: value.ImportPath(),
		},
		lessStoredPackageIdentity,
	)
}

func (table packageIdentityTable) fileReference(
	value identity.FileID,
) fileRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedFileIdentity,
		fileRef,
	](
		table.files,
		storedFileIdentity{
			owner: table.ownerReference(value.Owner()),
			rel:   value.Rel(),
		},
		lessStoredFileIdentity,
	)
}

func (table packageIdentityTable) spanReference(
	value identity.SpanID,
) spanRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedSpanIdentity,
		spanRef,
	](
		table.spans,
		storedSpanIdentity{
			file:  table.fileReference(value.File()),
			start: value.Start(),
			end:   value.End(),
		},
		lessStoredSpanIdentity,
	)
}

func (table packageIdentityTable) occurrenceReference(
	value identity.OccurrenceID,
) occurrenceRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedOccurrenceIdentity,
		occurrenceRef,
	](
		table.occurrences,
		storedOccurrenceIdentity{
			span: table.spanReference(value.Span()),
			kind: value.KindID(),
		},
		lessStoredOccurrenceIdentity,
	)
}

func (table packageIdentityTable) definitionReference(
	value identity.DefinitionID,
) definitionRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedDefinitionIdentity,
		definitionRef,
	](
		table.definitions,
		storedDefinitionIdentity{
			kind:      value.Kind(),
			root:      table.occurrenceReference(value.Root()),
			pkg:       table.packageReference(value.Package()),
			implicit:  value.ImplicitOp(),
			synthetic: value.SyntheticRole(),
			name:      value.SyntheticName(),
		},
		lessStoredDefinition,
	)
}

func (table packageIdentityTable) typeReference(
	value identity.SemanticTypeID,
) typeRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedTypeIdentity,
		typeRef,
	](
		table.types,
		storedTypeIdentity{digest: value.Digest()},
		lessStoredTypeIdentity,
	)
}

func (table packageIdentityTable) declarationReference(
	value identity.SemanticDeclarationID,
) declarationRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedDeclarationIdentity,
		declarationRef,
	](
		table.declarations,
		storedDeclarationIdentity{
			form:      value.Form(),
			pkg:       table.packageReference(value.Package()),
			ownerType: table.typeReference(value.OwnerType()),
			memberPkg: table.packageReference(
				value.MemberPackage(),
			),
			class:       value.Class(),
			name:        value.Name(),
			ordinal:     value.Ordinal(),
			predeclared: value.Predeclared(),
			owner: table.occurrenceReference(
				value.OwnerOccurrence(),
			),
			occurrence: table.occurrenceReference(
				value.Occurrence(),
			),
		},
		lessStoredDeclaration,
	)
}

func (table packageIdentityTable) bindingReference(
	value identity.SemanticBindingID,
) bindingRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedBindingIdentity,
		bindingRef,
	](
		table.bindings,
		storedBindingIdentity{
			owner: table.occurrenceReference(value.Owner()),
			declaration: table.occurrenceReference(
				value.Declaration(),
			),
			role:    value.Role(),
			ordinal: value.Ordinal(),
		},
		lessStoredBinding,
	)
}

func (table packageIdentityTable) operationReference(
	value identity.OperationID,
) operationRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedOperationIdentity,
		operationRef,
	](
		table.operations,
		storedOperationIdentity{
			definition: table.definitionReference(
				value.Definition(),
			),
			occurrence: table.occurrenceReference(
				value.Occurrence(),
			),
			implicit: value.ImplicitOp(),
			ordinal:  value.Ordinal(),
		},
		lessStoredOperation,
	)
}

func (table packageIdentityTable) unsupportedReference(
	value identity.UnsupportedID,
) unsupportedRef {
	if value.IsZero() {
		return 0
	}
	return storedIdentityReference[
		storedUnsupportedIdentity,
		unsupportedRef,
	](
		table.unsupported,
		storedUnsupportedIdentity{
			definition: table.definitionReference(
				value.Definition(),
			),
			occurrence: table.occurrenceReference(
				value.Occurrence(),
			),
		},
		lessStoredUnsupportedIdentity,
	)
}
