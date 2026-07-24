package semantic

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
)

func admitPackageIdentityTable(
	table packageIdentityTable,
) (admittedPackageIdentityTable, error) {
	if err := validatePackageIdentityTable(table); err != nil {
		return admittedPackageIdentityTable{}, err
	}
	return admittedPackageIdentityTable{table: table}, nil
}

func validatePackageIdentityTable(
	table packageIdentityTable,
) error {
	validators := []func() error{
		func() error {
			return validateIdentityComponents(
				"module",
				table.modules,
				lessStoredModuleIdentity,
				func(index int) bool {
					return !table.module(moduleRef(index + 1)).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"owner",
				table.owners,
				lessStoredOwnerIdentity,
				func(index int) bool {
					return !table.owner(ownerRef(index + 1)).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"package",
				table.packages,
				lessStoredPackageIdentity,
				func(index int) bool {
					return !table.packageID(packageRef(index + 1)).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"file",
				table.files,
				lessStoredFileIdentity,
				func(index int) bool {
					return !table.file(fileRef(index + 1)).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"span",
				table.spans,
				lessStoredSpanIdentity,
				func(index int) bool {
					return !table.span(spanRef(index + 1)).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"occurrence",
				table.occurrences,
				lessStoredOccurrenceIdentity,
				func(index int) bool {
					return !table.occurrence(
						occurrenceRef(index + 1),
					).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"definition",
				table.definitions,
				lessStoredDefinition,
				func(index int) bool {
					return !table.definition(
						definitionRef(index + 1),
					).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"type",
				table.types,
				lessStoredTypeIdentity,
				func(index int) bool {
					return !table.typeID(typeRef(index + 1)).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"declaration",
				table.declarations,
				lessStoredDeclaration,
				func(index int) bool {
					return !table.declaration(
						declarationRef(index + 1),
					).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"binding",
				table.bindings,
				lessStoredBinding,
				func(index int) bool {
					return !table.binding(
						bindingRef(index + 1),
					).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"operation",
				table.operations,
				lessStoredOperation,
				func(index int) bool {
					return !table.operation(
						operationRef(index + 1),
					).IsZero()
				},
			)
		},
		func() error {
			return validateIdentityComponents(
				"unsupported",
				table.unsupported,
				lessStoredUnsupportedIdentity,
				func(index int) bool {
					return !table.unsupportedID(
						unsupportedRef(index + 1),
					).IsZero()
				},
			)
		},
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}
	return nil
}

func validateIdentityComponents[Record any](
	name string,
	records []Record,
	less func(Record, Record) bool,
	valid func(int) bool,
) error {
	for index, record := range records {
		if index != 0 && !less(records[index-1], record) {
			return fmt.Errorf(
				"semantic %s identity table is not canonical at %d",
				name,
				index,
			)
		}
		if !valid(index) {
			return identitySealError(
				name,
				index,
				fmt.Errorf("component references are invalid"),
			)
		}
	}
	return nil
}

func identitySealError(name string, index int, err error) error {
	return fmt.Errorf(
		"seal semantic %s identity %d: %w",
		name,
		index+1,
		err,
	)
}

func projectOwnerIdentity(
	projection *packageIdentityProjection,
	record storedOwnerIdentity,
) (identity.Owner, error) {
	module := projection.projectModule(record.module)
	switch record.class {
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
			"owner class %d is invalid",
			record.class,
		)
	}
}

func projectDefinitionIdentity(
	projection *packageIdentityProjection,
	record storedDefinitionIdentity,
) (identity.DefinitionID, error) {
	root := projection.projectOccurrence(record.root)
	pkg := projection.projectPackage(record.pkg)
	switch {
	case record.kind.Source():
		if !pkg.IsZero() || record.implicit != 0 ||
			record.synthetic != 0 || record.name != "" {
			return identity.DefinitionID{}, fmt.Errorf(
				"source definition carries inactive fields",
			)
		}
		return identity.NewSourceDefinitionID(root, record.kind)
	case record.kind == identity.DefinitionImplicit &&
		record.implicit.Valid():
		if !root.IsZero() || record.synthetic != 0 ||
			record.name != "" {
			return identity.DefinitionID{}, fmt.Errorf(
				"implicit definition carries inactive fields",
			)
		}
		return identity.NewImplicitDefinitionID(pkg, record.implicit)
	case record.kind == identity.DefinitionImplicit &&
		record.synthetic.Valid():
		if !root.IsZero() || record.implicit != 0 {
			return identity.DefinitionID{}, fmt.Errorf(
				"synthetic definition carries inactive fields",
			)
		}
		return identity.NewSyntheticDefinitionID(
			pkg,
			record.synthetic,
			record.name,
		)
	default:
		return identity.DefinitionID{}, fmt.Errorf(
			"definition form is invalid",
		)
	}
}

func projectDeclarationIdentity(
	projection *packageIdentityProjection,
	record storedDeclarationIdentity,
) (identity.SemanticDeclarationID, error) {
	pkg := projection.projectPackage(record.pkg)
	ownerType := projection.projectType(record.ownerType)
	memberPackage := projection.projectPackage(record.memberPkg)
	owner := projection.projectOccurrence(record.owner)
	occurrence := projection.projectOccurrence(record.occurrence)
	switch record.form {
	case identity.SemanticDeclarationPackageObject:
		if !ownerType.IsZero() || !memberPackage.IsZero() ||
			record.ordinal != 0 || record.predeclared != 0 ||
			!owner.IsZero() || !occurrence.IsZero() {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"package declaration carries inactive fields",
			)
		}
		return identity.NewPackageDeclarationID(
			pkg,
			record.class,
			record.name,
		)
	case identity.SemanticDeclarationMember:
		if !pkg.IsZero() || record.predeclared != 0 ||
			!owner.IsZero() || !occurrence.IsZero() {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"member declaration carries inactive fields",
			)
		}
		return identity.NewMemberDeclarationID(
			ownerType,
			memberPackage,
			record.class,
			record.name,
			record.ordinal,
		)
	case identity.SemanticDeclarationPredeclared:
		if !pkg.IsZero() || !ownerType.IsZero() ||
			!memberPackage.IsZero() || record.name != "" ||
			record.ordinal != 0 || !owner.IsZero() ||
			!occurrence.IsZero() {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"predeclared declaration carries inactive fields",
			)
		}
		return identity.NewPredeclaredDeclarationID(
			record.predeclared,
			record.class,
		)
	case identity.SemanticDeclarationOccurrence:
		if !pkg.IsZero() || !ownerType.IsZero() ||
			!memberPackage.IsZero() || record.predeclared != 0 {
			return identity.SemanticDeclarationID{}, fmt.Errorf(
				"occurrence declaration carries inactive fields",
			)
		}
		return identity.NewOccurrenceDeclarationID(
			owner,
			occurrence,
			record.class,
			record.name,
			record.ordinal,
		)
	default:
		return identity.SemanticDeclarationID{}, fmt.Errorf(
			"declaration form %d is invalid",
			record.form,
		)
	}
}

func projectOperationIdentity(
	projection *packageIdentityProjection,
	record storedOperationIdentity,
) (identity.OperationID, error) {
	definition := projection.projectDefinition(record.definition)
	occurrence := projection.projectOccurrence(record.occurrence)
	if record.implicit.Valid() {
		if !occurrence.IsZero() {
			return identity.OperationID{}, fmt.Errorf(
				"implicit operation carries occurrence",
			)
		}
		return identity.NewImplicitOperationID(
			definition,
			record.implicit,
			record.ordinal,
		)
	}
	if record.implicit != 0 || record.ordinal != 0 {
		return identity.OperationID{}, fmt.Errorf(
			"source operation carries implicit fields",
		)
	}
	return identity.NewOperationID(definition, occurrence)
}

func lessStoredModuleIdentity(
	left storedModuleIdentity,
	right storedModuleIdentity,
) bool {
	if left.path != right.path {
		return left.path < right.path
	}
	return left.version < right.version
}

func lessStoredOwnerIdentity(
	left storedOwnerIdentity,
	right storedOwnerIdentity,
) bool {
	if left.class != right.class {
		return left.class < right.class
	}
	return left.module < right.module
}

func lessStoredPackageIdentity(
	left storedPackageIdentity,
	right storedPackageIdentity,
) bool {
	if left.owner != right.owner {
		return left.owner < right.owner
	}
	return left.importPath < right.importPath
}

func lessStoredFileIdentity(
	left storedFileIdentity,
	right storedFileIdentity,
) bool {
	if left.owner != right.owner {
		return left.owner < right.owner
	}
	return left.rel < right.rel
}

func lessStoredSpanIdentity(
	left storedSpanIdentity,
	right storedSpanIdentity,
) bool {
	if left.file != right.file {
		return left.file < right.file
	}
	if left.start != right.start {
		return left.start < right.start
	}
	return left.end < right.end
}

func lessStoredOccurrenceIdentity(
	left storedOccurrenceIdentity,
	right storedOccurrenceIdentity,
) bool {
	if left.span != right.span {
		return left.span < right.span
	}
	return left.kind < right.kind
}

func lessStoredTypeIdentity(
	left storedTypeIdentity,
	right storedTypeIdentity,
) bool {
	return left.digest < right.digest
}

func lessStoredUnsupportedIdentity(
	left storedUnsupportedIdentity,
	right storedUnsupportedIdentity,
) bool {
	if left.definition != right.definition {
		return left.definition < right.definition
	}
	return left.occurrence < right.occurrence
}
