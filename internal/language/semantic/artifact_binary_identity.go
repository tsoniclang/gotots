package semantic

import "github.com/tsoniclang/gotots/internal/identity"

func writeBinaryIdentityTables(
	encoder *binaryShardEncoder,
	table packageIdentityTable,
) {
	writeBinaryRecords(
		encoder,
		table.modules,
		func(encoder *binaryShardEncoder, record storedModuleIdentity) {
			encoder.text(record.path)
			encoder.text(record.version)
		},
	)
	writeBinaryRecords(
		encoder,
		table.owners,
		func(encoder *binaryShardEncoder, record storedOwnerIdentity) {
			encoder.unsigned(uint64(record.class))
			encoder.unsigned(uint64(record.module))
		},
	)
	writeBinaryRecords(
		encoder,
		table.packages,
		func(encoder *binaryShardEncoder, record storedPackageIdentity) {
			encoder.unsigned(uint64(record.owner))
			encoder.text(record.importPath)
		},
	)
	writeBinaryRecords(
		encoder,
		table.files,
		func(encoder *binaryShardEncoder, record storedFileIdentity) {
			encoder.unsigned(uint64(record.owner))
			encoder.text(record.rel)
		},
	)
	writeBinaryRecords(
		encoder,
		table.spans,
		func(encoder *binaryShardEncoder, record storedSpanIdentity) {
			encoder.unsigned(uint64(record.file))
			encoder.signed(int64(record.start))
			encoder.signed(int64(record.end))
		},
	)
	writeBinaryRecords(
		encoder,
		table.occurrences,
		func(encoder *binaryShardEncoder, record storedOccurrenceIdentity) {
			encoder.unsigned(uint64(record.span))
			encoder.unsigned(uint64(record.kind))
		},
	)
	writeBinaryRecords(
		encoder,
		table.definitions,
		func(encoder *binaryShardEncoder, record storedDefinitionIdentity) {
			encoder.unsigned(uint64(record.kind))
			encoder.unsigned(uint64(record.root))
			encoder.unsigned(uint64(record.pkg))
			encoder.unsigned(uint64(record.implicit))
			encoder.unsigned(uint64(record.synthetic))
			encoder.text(record.name)
		},
	)
	writeBinaryRecords(
		encoder,
		table.types,
		func(encoder *binaryShardEncoder, record storedTypeIdentity) {
			encoder.text(record.digest)
		},
	)
	writeBinaryRecords(
		encoder,
		table.declarations,
		func(encoder *binaryShardEncoder, record storedDeclarationIdentity) {
			encoder.unsigned(uint64(record.form))
			encoder.unsigned(uint64(record.pkg))
			encoder.unsigned(uint64(record.ownerType))
			encoder.unsigned(uint64(record.memberPkg))
			encoder.unsigned(uint64(record.class))
			encoder.text(record.name)
			encoder.signed(int64(record.ordinal))
			encoder.unsigned(uint64(record.predeclared))
			encoder.unsigned(uint64(record.owner))
			encoder.unsigned(uint64(record.occurrence))
		},
	)
	writeBinaryRecords(
		encoder,
		table.bindings,
		func(encoder *binaryShardEncoder, record storedBindingIdentity) {
			encoder.unsigned(uint64(record.owner))
			encoder.unsigned(uint64(record.declaration))
			encoder.unsigned(uint64(record.role))
			encoder.signed(int64(record.ordinal))
		},
	)
	writeBinaryRecords(
		encoder,
		table.operations,
		func(encoder *binaryShardEncoder, record storedOperationIdentity) {
			encoder.unsigned(uint64(record.definition))
			encoder.unsigned(uint64(record.occurrence))
			encoder.unsigned(uint64(record.implicit))
			encoder.signed(int64(record.ordinal))
		},
	)
	writeBinaryRecords(
		encoder,
		table.unsupported,
		func(encoder *binaryShardEncoder, record storedUnsupportedIdentity) {
			encoder.unsigned(uint64(record.definition))
			encoder.unsigned(uint64(record.occurrence))
		},
	)
}

func readBinaryIdentityTables(
	decoder *binaryShardDecoder,
) (packageIdentityTable, error) {
	var (
		components packageIdentityComponents
		err        error
	)
	components.modules, err = readBinaryRecords(
		decoder,
		"module identities",
		func(decoder *binaryShardDecoder) (storedModuleIdentity, error) {
			path, readErr := decoder.text("module path")
			if readErr != nil {
				return storedModuleIdentity{}, readErr
			}
			version, readErr := decoder.text("module version")
			return storedModuleIdentity{
				path: path, version: version,
			}, readErr
		},
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.owners, err = readBinaryRecords(
		decoder,
		"owner identities",
		func(decoder *binaryShardDecoder) (storedOwnerIdentity, error) {
			class, readErr := readUnsignedAs[identity.OwnerClass](
				decoder, "owner class",
			)
			if readErr != nil {
				return storedOwnerIdentity{}, readErr
			}
			module, readErr := readIdentityReference[moduleRef](
				decoder, "owner module",
			)
			return storedOwnerIdentity{
				class: class, module: module,
			}, readErr
		},
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.packages, err = readBinaryRecords(
		decoder,
		"package identities",
		func(decoder *binaryShardDecoder) (storedPackageIdentity, error) {
			owner, readErr := readIdentityReference[ownerRef](
				decoder, "package owner",
			)
			if readErr != nil {
				return storedPackageIdentity{}, readErr
			}
			importPath, readErr := decoder.text("package import path")
			return storedPackageIdentity{
				owner: owner, importPath: importPath,
			}, readErr
		},
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.files, err = readBinaryRecords(
		decoder,
		"file identities",
		func(decoder *binaryShardDecoder) (storedFileIdentity, error) {
			owner, readErr := readIdentityReference[ownerRef](
				decoder, "file owner",
			)
			if readErr != nil {
				return storedFileIdentity{}, readErr
			}
			relative, readErr := decoder.text("file relative path")
			return storedFileIdentity{
				owner: owner, rel: relative,
			}, readErr
		},
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.spans, err = readBinaryRecords(
		decoder,
		"span identities",
		readBinarySpanIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.occurrences, err = readBinaryRecords(
		decoder,
		"occurrence identities",
		readBinaryOccurrenceIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.definitions, err = readBinaryRecords(
		decoder,
		"definition identities",
		readBinaryDefinitionIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.types, err = readBinaryRecords(
		decoder,
		"type identities",
		func(decoder *binaryShardDecoder) (storedTypeIdentity, error) {
			digest, readErr := decoder.text("type digest")
			return storedTypeIdentity{digest: digest}, readErr
		},
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.declarations, err = readBinaryRecords(
		decoder,
		"declaration identities",
		readBinaryDeclarationIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.bindings, err = readBinaryRecords(
		decoder,
		"binding identities",
		readBinaryBindingIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.operations, err = readBinaryRecords(
		decoder,
		"operation identities",
		readBinaryOperationIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	components.unsupported, err = readBinaryRecords(
		decoder,
		"unsupported identities",
		readBinaryUnsupportedIdentity,
	)
	if err != nil {
		return packageIdentityTable{}, err
	}
	return packageIdentityTable{
		packageIdentityComponents: components,
	}, nil
}

func readBinarySpanIdentity(
	decoder *binaryShardDecoder,
) (storedSpanIdentity, error) {
	file, err := readIdentityReference[fileRef](decoder, "span file")
	if err != nil {
		return storedSpanIdentity{}, err
	}
	start, err := readSignedInt(decoder, "span start")
	if err != nil {
		return storedSpanIdentity{}, err
	}
	end, err := readSignedInt(decoder, "span end")
	return storedSpanIdentity{file: file, start: start, end: end}, err
}

func readBinaryOccurrenceIdentity(
	decoder *binaryShardDecoder,
) (storedOccurrenceIdentity, error) {
	span, err := readIdentityReference[spanRef](
		decoder, "occurrence span",
	)
	if err != nil {
		return storedOccurrenceIdentity{}, err
	}
	kind, err := readUnsignedAs[uint16](decoder, "occurrence kind")
	return storedOccurrenceIdentity{span: span, kind: kind}, err
}

func readBinaryDefinitionIdentity(
	decoder *binaryShardDecoder,
) (storedDefinitionIdentity, error) {
	kind, err := readUnsignedAs[identity.DefinitionKind](
		decoder, "definition identity kind",
	)
	if err != nil {
		return storedDefinitionIdentity{}, err
	}
	root, err := readIdentityReference[occurrenceRef](
		decoder, "definition identity root",
	)
	if err != nil {
		return storedDefinitionIdentity{}, err
	}
	pkg, err := readIdentityReference[packageRef](
		decoder, "definition identity package",
	)
	if err != nil {
		return storedDefinitionIdentity{}, err
	}
	implicit, err := readUnsignedAs[identity.ImplicitDefinitionOp](
		decoder, "definition identity implicit operation",
	)
	if err != nil {
		return storedDefinitionIdentity{}, err
	}
	synthetic, err := readUnsignedAs[identity.SyntheticDefinitionRole](
		decoder, "definition identity synthetic role",
	)
	if err != nil {
		return storedDefinitionIdentity{}, err
	}
	name, err := decoder.text("definition identity name")
	return storedDefinitionIdentity{
		kind: kind, root: root, pkg: pkg,
		implicit: implicit, synthetic: synthetic, name: name,
	}, err
}

func readBinaryDeclarationIdentity(
	decoder *binaryShardDecoder,
) (storedDeclarationIdentity, error) {
	form, err := readUnsignedAs[identity.SemanticDeclarationForm](
		decoder, "declaration identity form",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	pkg, err := readIdentityReference[packageRef](
		decoder, "declaration identity package",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	ownerType, err := readIdentityReference[typeRef](
		decoder, "declaration identity owner type",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	memberPackage, err := readIdentityReference[packageRef](
		decoder, "declaration identity member package",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	class, err := readUnsignedAs[identity.SemanticObjectClass](
		decoder, "declaration identity class",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	name, err := decoder.text("declaration identity name")
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	ordinal, err := readSignedInt(decoder, "declaration identity ordinal")
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	predeclared, err := readUnsignedAs[uint16](
		decoder, "declaration identity predeclared",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	owner, err := readIdentityReference[occurrenceRef](
		decoder, "declaration identity owner occurrence",
	)
	if err != nil {
		return storedDeclarationIdentity{}, err
	}
	occurrence, err := readIdentityReference[occurrenceRef](
		decoder, "declaration identity occurrence",
	)
	return storedDeclarationIdentity{
		form: form, pkg: pkg, ownerType: ownerType,
		memberPkg: memberPackage, class: class, name: name,
		ordinal: ordinal, predeclared: predeclared,
		owner: owner, occurrence: occurrence,
	}, err
}

func readBinaryBindingIdentity(
	decoder *binaryShardDecoder,
) (storedBindingIdentity, error) {
	owner, err := readIdentityReference[occurrenceRef](
		decoder, "binding identity owner",
	)
	if err != nil {
		return storedBindingIdentity{}, err
	}
	declaration, err := readIdentityReference[occurrenceRef](
		decoder, "binding identity declaration",
	)
	if err != nil {
		return storedBindingIdentity{}, err
	}
	role, err := readUnsignedAs[identity.SemanticBindingRole](
		decoder, "binding identity role",
	)
	if err != nil {
		return storedBindingIdentity{}, err
	}
	ordinal, err := readSignedInt(decoder, "binding identity ordinal")
	return storedBindingIdentity{
		owner: owner, declaration: declaration,
		role: role, ordinal: ordinal,
	}, err
}

func readBinaryOperationIdentity(
	decoder *binaryShardDecoder,
) (storedOperationIdentity, error) {
	definition, err := readIdentityReference[definitionRef](
		decoder, "operation identity definition",
	)
	if err != nil {
		return storedOperationIdentity{}, err
	}
	occurrence, err := readIdentityReference[occurrenceRef](
		decoder, "operation identity occurrence",
	)
	if err != nil {
		return storedOperationIdentity{}, err
	}
	implicit, err := readUnsignedAs[identity.ImplicitDefinitionOp](
		decoder, "operation identity implicit",
	)
	if err != nil {
		return storedOperationIdentity{}, err
	}
	ordinal, err := readSignedInt(decoder, "operation identity ordinal")
	return storedOperationIdentity{
		definition: definition, occurrence: occurrence,
		implicit: implicit, ordinal: ordinal,
	}, err
}

func readBinaryUnsupportedIdentity(
	decoder *binaryShardDecoder,
) (storedUnsupportedIdentity, error) {
	definition, err := readIdentityReference[definitionRef](
		decoder, "unsupported identity definition",
	)
	if err != nil {
		return storedUnsupportedIdentity{}, err
	}
	occurrence, err := readIdentityReference[occurrenceRef](
		decoder, "unsupported identity occurrence",
	)
	return storedUnsupportedIdentity{
		definition: definition, occurrence: occurrence,
	}, err
}
