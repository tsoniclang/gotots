package semantic

import "io"

func writePackageIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "packages", len(table.packages),
		func(index int) (wirePackageIdentity, error) {
			value := table.packages[index]
			owner, err := encoder.owner(value.owner)
			return wirePackageIdentity{
				Owner: owner, ImportPath: value.importPath,
			}, err
		},
	)
}

func writeFileIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "files", len(table.files),
		func(index int) (wireFileIdentity, error) {
			value := table.files[index]
			owner, err := encoder.owner(value.owner)
			return wireFileIdentity{
				Owner: owner, Rel: value.rel,
			}, err
		},
	)
}

func writeSpanIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "spans", len(table.spans),
		func(index int) (wireSpanIdentity, error) {
			value := table.spans[index]
			file, err := encoder.file(value.file)
			return wireSpanIdentity{
				File: file, Start: value.start, End: value.end,
			}, err
		},
	)
}

func writeOccurrenceIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "occurrences", len(table.occurrences),
		func(index int) (wireOccurrenceIdentity, error) {
			value := table.occurrences[index]
			span, err := encoder.span(value.span)
			return wireOccurrenceIdentity{
				Span: span, Kind: value.kind,
			}, err
		},
	)
}

func writeDefinitionIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "definitions", len(table.definitions),
		func(index int) (wireDefinitionIdentity, error) {
			value := table.definitions[index]
			root, err := encoder.occurrence(value.root)
			if err != nil {
				return wireDefinitionIdentity{}, err
			}
			pkg, err := encoder.packageID(value.pkg)
			return wireDefinitionIdentity{
				Kind:      uint8(value.kind),
				Root:      root,
				Package:   pkg,
				Implicit:  uint8(value.implicit),
				Synthetic: uint8(value.synthetic),
				Name:      value.name,
			}, err
		},
	)
}

func writeTypeIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "types", len(table.types),
		func(index int) (wireTypeIdentity, error) {
			return wireTypeIdentity{
				Digest: table.types[index].digest,
			}, nil
		},
	)
}

func writeDeclarationIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "declarations", len(table.declarations),
		func(index int) (wireDeclarationIdentity, error) {
			value := table.declarations[index]
			pkg, err := encoder.packageID(value.pkg)
			if err != nil {
				return wireDeclarationIdentity{}, err
			}
			ownerType, err := encoder.typeID(value.ownerType)
			if err != nil {
				return wireDeclarationIdentity{}, err
			}
			memberPkg, err := encoder.packageID(value.memberPkg)
			if err != nil {
				return wireDeclarationIdentity{}, err
			}
			owner, err := encoder.occurrence(value.owner)
			if err != nil {
				return wireDeclarationIdentity{}, err
			}
			occurrence, err := encoder.occurrence(
				value.occurrence,
			)
			return wireDeclarationIdentity{
				Form:        uint8(value.form),
				Package:     pkg,
				OwnerType:   ownerType,
				MemberPkg:   memberPkg,
				Class:       uint8(value.class),
				Name:        value.name,
				Ordinal:     value.ordinal,
				Predeclared: value.predeclared,
				Owner:       owner,
				Occurrence:  occurrence,
			}, err
		},
	)
}

func writeBindingIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "bindings", len(table.bindings),
		func(index int) (wireBindingIdentity, error) {
			value := table.bindings[index]
			owner, err := encoder.occurrence(value.owner)
			if err != nil {
				return wireBindingIdentity{}, err
			}
			declaration, err := encoder.occurrence(
				value.declaration,
			)
			return wireBindingIdentity{
				Owner:       owner,
				Declaration: declaration,
				Role:        uint8(value.role),
				Ordinal:     value.ordinal,
			}, err
		},
	)
}

func writeOperationIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "operations", len(table.operations),
		func(index int) (wireOperationIdentity, error) {
			value := table.operations[index]
			definition, err := encoder.definition(
				value.definition,
			)
			if err != nil {
				return wireOperationIdentity{}, err
			}
			occurrence, err := encoder.occurrence(
				value.occurrence,
			)
			return wireOperationIdentity{
				Definition: definition,
				Occurrence: occurrence,
				Implicit:   uint8(value.implicit),
				Ordinal:    value.ordinal,
			}, err
		},
	)
}

func writeUnsupportedIdentities(
	output io.Writer,
	encoder wireIdentityEncoder,
) error {
	table := encoder.table
	return writeWireArray(
		output, "unsupported", len(table.unsupported),
		func(index int) (wireUnsupportedIdentity, error) {
			value := table.unsupported[index]
			definition, err := encoder.definition(
				value.definition,
			)
			if err != nil {
				return wireUnsupportedIdentity{}, err
			}
			occurrence, err := encoder.occurrence(
				value.occurrence,
			)
			return wireUnsupportedIdentity{
				Definition: definition,
				Occurrence: occurrence,
			}, err
		},
	)
}
