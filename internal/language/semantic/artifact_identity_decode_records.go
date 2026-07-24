package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func mappedWireIdentity[
	Wire ~uint64,
	Reference ~uint64,
](
	name string,
	value Wire,
	table wireIdentityTable[Reference],
) (Reference, error) {
	var zero Reference
	if value == 0 {
		return zero, nil
	}
	index := uint64(value)
	if index >= uint64(len(table.references)) {
		return zero, fmt.Errorf(
			"semantic wire %s reference %d is invalid",
			name,
			value,
		)
	}
	if int(index) >= len(table.used) {
		return zero, fmt.Errorf(
			"semantic wire %s usage index %d is invalid",
			name,
			value,
		)
	}
	table.used[index] = true
	return table.references[index], nil
}

func (decoder wireIdentityDecoder) projector() *packageIdentityProjection {
	decoder.projection.table = decoder.builder.projectionTable()
	return decoder.projection
}

func (decoder wireIdentityDecoder) moduleRef(
	value wireModuleReference,
) (moduleRef, error) {
	return mappedWireIdentity(
		"module", value, decoder.modules,
	)
}

func (decoder wireIdentityDecoder) ownerRef(
	value wireOwnerReference,
) (ownerRef, error) {
	return mappedWireIdentity(
		"owner", value, decoder.owners,
	)
}

func (decoder wireIdentityDecoder) packageRef(
	value wirePackageReference,
) (packageRef, error) {
	return mappedWireIdentity(
		"package", value, decoder.packages,
	)
}

func (decoder wireIdentityDecoder) fileRef(
	value wireFileReference,
) (fileRef, error) {
	return mappedWireIdentity(
		"file", value, decoder.files,
	)
}

func (decoder wireIdentityDecoder) spanRef(
	value wireSpanReference,
) (spanRef, error) {
	return mappedWireIdentity(
		"span", value, decoder.spans,
	)
}

func (decoder wireIdentityDecoder) occurrenceRef(
	value wireOccurrenceReference,
) (occurrenceRef, error) {
	return mappedWireIdentity(
		"occurrence", value, decoder.occurrences,
	)
}

func (decoder wireIdentityDecoder) definitionRef(
	value wireDefinitionReference,
) (definitionRef, error) {
	return mappedWireIdentity(
		"definition", value, decoder.definitions,
	)
}

func (decoder wireIdentityDecoder) typeRef(
	value wireTypeReference,
) (typeRef, error) {
	return mappedWireIdentity(
		"type", value, decoder.types,
	)
}

func (decoder wireIdentityDecoder) declarationRef(
	value wireDeclarationReference,
) (declarationRef, error) {
	return mappedWireIdentity(
		"declaration", value, decoder.declarations,
	)
}

func (decoder wireIdentityDecoder) bindingRef(
	value wireBindingReference,
) (bindingRef, error) {
	return mappedWireIdentity(
		"binding", value, decoder.bindings,
	)
}

func (decoder wireIdentityDecoder) operationRef(
	value wireOperationReference,
) (operationRef, error) {
	return mappedWireIdentity(
		"operation", value, decoder.operations,
	)
}

func (decoder wireIdentityDecoder) unsupportedRef(
	value wireUnsupportedReference,
) (unsupportedRef, error) {
	return mappedWireIdentity(
		"unsupported", value, decoder.unsupported,
	)
}

func validateWireIdentityUsage(name string, used []bool) error {
	for index := 1; index < len(used); index++ {
		if !used[index] {
			return fmt.Errorf(
				"semantic wire %s identity %d is unreferenced",
				name,
				index,
			)
		}
	}
	return nil
}

func (decoder wireIdentityDecoder) validateUsage() error {
	checks := []struct {
		name string
		used []bool
	}{
		{name: "modules", used: decoder.modules.used},
		{name: "owners", used: decoder.owners.used},
		{name: "packages", used: decoder.packages.used},
		{name: "files", used: decoder.files.used},
		{name: "spans", used: decoder.spans.used},
		{name: "occurrences", used: decoder.occurrences.used},
		{name: "definitions", used: decoder.definitions.used},
		{name: "types", used: decoder.types.used},
		{name: "declarations", used: decoder.declarations.used},
		{name: "bindings", used: decoder.bindings.used},
		{name: "operations", used: decoder.operations.used},
		{name: "unsupported", used: decoder.unsupported.used},
	}
	for _, check := range checks {
		if err := validateWireIdentityUsage(
			check.name, check.used,
		); err != nil {
			return err
		}
	}
	return nil
}

func (decoder wireIdentityDecoder) module(
	value wireModuleReference,
) (identity.ModuleID, error) {
	reference, err := decoder.moduleRef(value)
	if err != nil {
		return identity.ModuleID{}, err
	}
	return decoder.projector().module(reference), nil
}

func (decoder wireIdentityDecoder) owner(
	value wireOwnerReference,
) (identity.Owner, error) {
	reference, err := decoder.ownerRef(value)
	if err != nil {
		return identity.Owner{}, err
	}
	return decoder.projector().owner(reference), nil
}

func (decoder wireIdentityDecoder) packageID(
	value wirePackageReference,
) (identity.PackageID, error) {
	reference, err := decoder.packageRef(value)
	if err != nil {
		return identity.PackageID{}, err
	}
	return decoder.projector().packageID(reference), nil
}

func (decoder wireIdentityDecoder) file(
	value wireFileReference,
) (identity.FileID, error) {
	reference, err := decoder.fileRef(value)
	if err != nil {
		return identity.FileID{}, err
	}
	return decoder.projector().file(reference), nil
}

func (decoder wireIdentityDecoder) span(
	value wireSpanReference,
) (identity.SpanID, error) {
	reference, err := decoder.spanRef(value)
	if err != nil {
		return identity.SpanID{}, err
	}
	return decoder.projector().span(reference), nil
}

func (decoder wireIdentityDecoder) occurrence(
	value wireOccurrenceReference,
) (identity.OccurrenceID, error) {
	reference, err := decoder.occurrenceRef(value)
	if err != nil {
		return identity.OccurrenceID{}, err
	}
	return decoder.projector().occurrence(reference), nil
}

func (decoder wireIdentityDecoder) definition(
	value wireDefinitionReference,
) (identity.DefinitionID, error) {
	reference, err := decoder.definitionRef(value)
	if err != nil {
		return identity.DefinitionID{}, err
	}
	return decoder.projector().definition(reference), nil
}

func (decoder wireIdentityDecoder) typeID(
	value wireTypeReference,
) (identity.SemanticTypeID, error) {
	reference, err := decoder.typeRef(value)
	if err != nil {
		return identity.SemanticTypeID{}, err
	}
	return decoder.projector().typeID(reference), nil
}

func (decoder wireIdentityDecoder) declaration(
	value wireDeclarationReference,
) (identity.SemanticDeclarationID, error) {
	reference, err := decoder.declarationRef(value)
	if err != nil {
		return identity.SemanticDeclarationID{}, err
	}
	return decoder.projector().declaration(reference), nil
}

func (decoder wireIdentityDecoder) binding(
	value wireBindingReference,
) (identity.SemanticBindingID, error) {
	reference, err := decoder.bindingRef(value)
	if err != nil {
		return identity.SemanticBindingID{}, err
	}
	return decoder.projector().binding(reference), nil
}

func (decoder wireIdentityDecoder) operation(
	value wireOperationReference,
) (identity.OperationID, error) {
	reference, err := decoder.operationRef(value)
	if err != nil {
		return identity.OperationID{}, err
	}
	return decoder.projector().operation(reference), nil
}

func (decoder wireIdentityDecoder) unsupportedID(
	value wireUnsupportedReference,
) (identity.UnsupportedID, error) {
	reference, err := decoder.unsupportedRef(value)
	if err != nil {
		return identity.UnsupportedID{}, err
	}
	return decoder.projector().unsupportedID(reference), nil
}

func (decoder wireIdentityDecoder) decodeOwner(
	record wireOwnerIdentity,
) (identity.Owner, error) {
	class := identity.OwnerClass(record.Class)
	module, err := decoder.module(record.Module)
	if err != nil {
		return identity.Owner{}, err
	}
	switch class {
	case identity.OwnerModule:
		return identity.NewModuleOwner(module)
	case identity.OwnerStandardLibrary:
		if !module.IsZero() {
			return identity.Owner{}, fmt.Errorf(
				"standard-library owner carries a module",
			)
		}
		return identity.StandardLibraryOwner(), nil
	case identity.OwnerToolchain:
		if !module.IsZero() {
			return identity.Owner{}, fmt.Errorf(
				"toolchain owner carries a module",
			)
		}
		return identity.ToolchainOwner(), nil
	case identity.OwnerLanguagePseudo:
		if !module.IsZero() {
			return identity.Owner{}, fmt.Errorf(
				"language owner carries a module",
			)
		}
		return identity.LanguagePseudoOwner(), nil
	default:
		return identity.Owner{}, fmt.Errorf(
			"semantic wire owner class %d is invalid", record.Class,
		)
	}
}

func (decoder wireIdentityDecoder) decodePackage(
	record wirePackageIdentity,
) (identity.PackageID, error) {
	owner, err := decoder.owner(record.Owner)
	if err != nil {
		return identity.PackageID{}, err
	}
	return identity.NewPackageID(owner, record.ImportPath)
}

func (decoder wireIdentityDecoder) decodeFile(
	record wireFileIdentity,
) (identity.FileID, error) {
	owner, err := decoder.owner(record.Owner)
	if err != nil {
		return identity.FileID{}, err
	}
	return identity.NewFileID(owner, record.Rel)
}

func (decoder wireIdentityDecoder) decodeSpan(
	record wireSpanIdentity,
) (identity.SpanID, error) {
	file, err := decoder.file(record.File)
	if err != nil {
		return identity.SpanID{}, err
	}
	return identity.NewSpanID(file, record.Start, record.End)
}

func (decoder wireIdentityDecoder) decodeOccurrence(
	record wireOccurrenceIdentity,
) (identity.OccurrenceID, error) {
	span, err := decoder.span(record.Span)
	if err != nil {
		return identity.OccurrenceID{}, err
	}
	return identity.NewOccurrenceID(span, record.Kind)
}
