package semantic

import (
	"fmt"
	"sort"
)

type packageIdentityRemap struct {
	modules      []uint64
	owners       []uint64
	packages     []uint64
	files        []uint64
	spans        []uint64
	occurrences  []uint64
	definitions  []uint64
	types        []uint64
	declarations []uint64
	bindings     []uint64
	operations   []uint64
	unsupported  []uint64
}

func canonicalizeComponents[Record any](
	records []Record,
	less func(Record, Record) bool,
) ([]Record, []uint64) {
	order := make([]int, len(records))
	for index := range order {
		order[index] = index
	}
	sort.Slice(order, func(left, right int) bool {
		return less(records[order[left]], records[order[right]])
	})
	remap := make([]uint64, len(records)+1)
	for newIndex, oldIndex := range order {
		remap[oldIndex+1] = uint64(newIndex + 1)
	}
	visited := make([]bool, len(records))
	for start := range records {
		if visited[start] {
			continue
		}
		current := start
		value := records[start]
		for {
			visited[current] = true
			source := order[current]
			if source == start {
				records[current] = value
				break
			}
			records[current] = records[source]
			current = source
		}
	}
	return records, remap
}

func remapReference[Ref ~uint64](
	reference Ref,
	remap []uint64,
) (Ref, error) {
	if reference == 0 {
		return 0, nil
	}
	index := uint64(reference)
	if index >= uint64(len(remap)) || remap[index] == 0 {
		return 0, fmt.Errorf(
			"semantic package identity reference %d is invalid",
			reference,
		)
	}
	return Ref(remap[index]), nil
}

func (builder *packageIdentityBuilder) seal() (
	packageIdentityTable,
	packageIdentityRemap,
	error,
) {
	var components packageIdentityComponents
	var remap packageIdentityRemap
	components.modules, remap.modules = canonicalizeComponents(
		builder.modules.records,
		lessStoredModuleIdentity,
	)
	for index := range builder.owners.records {
		record := &builder.owners.records[index]
		var err error
		record.module, err = remapReference(record.module, remap.modules)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.owners, remap.owners = canonicalizeComponents(
		builder.owners.records,
		lessStoredOwnerIdentity,
	)
	for index := range builder.packages.records {
		record := &builder.packages.records[index]
		var err error
		record.owner, err = remapReference(record.owner, remap.owners)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.packages, remap.packages = canonicalizeComponents(
		builder.packages.records,
		lessStoredPackageIdentity,
	)
	for index := range builder.files.records {
		record := &builder.files.records[index]
		var err error
		record.owner, err = remapReference(record.owner, remap.owners)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.files, remap.files = canonicalizeComponents(
		builder.files.records,
		lessStoredFileIdentity,
	)
	for index := range builder.spans.records {
		record := &builder.spans.records[index]
		var err error
		record.file, err = remapReference(record.file, remap.files)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.spans, remap.spans = canonicalizeComponents(
		builder.spans.records,
		lessStoredSpanIdentity,
	)
	for index := range builder.occurrences.records {
		record := &builder.occurrences.records[index]
		var err error
		record.span, err = remapReference(record.span, remap.spans)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.occurrences, remap.occurrences = canonicalizeComponents(
		builder.occurrences.records,
		lessStoredOccurrenceIdentity,
	)
	for index := range builder.definitions.records {
		record := &builder.definitions.records[index]
		var err error
		record.root, err = remapReference(
			record.root, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.pkg, err = remapReference(record.pkg, remap.packages)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.definitions, remap.definitions = canonicalizeComponents(
		builder.definitions.records,
		lessStoredDefinition,
	)
	components.types, remap.types = canonicalizeComponents(
		builder.types.records,
		lessStoredTypeIdentity,
	)
	for index := range builder.declarations.records {
		record := &builder.declarations.records[index]
		var err error
		record.pkg, err = remapReference(record.pkg, remap.packages)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.ownerType, err = remapReference(
			record.ownerType, remap.types,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.memberPkg, err = remapReference(
			record.memberPkg, remap.packages,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.owner, err = remapReference(
			record.owner, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.occurrence, err = remapReference(
			record.occurrence, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.declarations, remap.declarations = canonicalizeComponents(
		builder.declarations.records,
		lessStoredDeclaration,
	)
	for index := range builder.bindings.records {
		record := &builder.bindings.records[index]
		var err error
		record.owner, err = remapReference(
			record.owner, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.declaration, err = remapReference(
			record.declaration, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.bindings, remap.bindings = canonicalizeComponents(
		builder.bindings.records,
		lessStoredBinding,
	)
	for index := range builder.operations.records {
		record := &builder.operations.records[index]
		var err error
		record.definition, err = remapReference(
			record.definition, remap.definitions,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.occurrence, err = remapReference(
			record.occurrence, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.operations, remap.operations = canonicalizeComponents(
		builder.operations.records,
		lessStoredOperation,
	)
	for index := range builder.unsupported.records {
		record := &builder.unsupported.records[index]
		var err error
		record.definition, err = remapReference(
			record.definition, remap.definitions,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
		record.occurrence, err = remapReference(
			record.occurrence, remap.occurrences,
		)
		if err != nil {
			return packageIdentityTable{}, packageIdentityRemap{}, err
		}
	}
	components.unsupported, remap.unsupported = canonicalizeComponents(
		builder.unsupported.records,
		lessStoredUnsupportedIdentity,
	)
	return packageIdentityTable{
		packageIdentityComponents: components,
	}, remap, nil
}

func lessStoredDefinition(
	left storedDefinitionIdentity,
	right storedDefinitionIdentity,
) bool {
	if left.kind != right.kind {
		return left.kind < right.kind
	}
	if left.root != right.root {
		return left.root < right.root
	}
	if left.pkg != right.pkg {
		return left.pkg < right.pkg
	}
	if left.implicit != right.implicit {
		return left.implicit < right.implicit
	}
	if left.synthetic != right.synthetic {
		return left.synthetic < right.synthetic
	}
	return left.name < right.name
}

func lessStoredDeclaration(
	left storedDeclarationIdentity,
	right storedDeclarationIdentity,
) bool {
	switch {
	case left.form != right.form:
		return left.form < right.form
	case left.pkg != right.pkg:
		return left.pkg < right.pkg
	case left.ownerType != right.ownerType:
		return left.ownerType < right.ownerType
	case left.memberPkg != right.memberPkg:
		return left.memberPkg < right.memberPkg
	case left.class != right.class:
		return left.class < right.class
	case left.name != right.name:
		return left.name < right.name
	case left.ordinal != right.ordinal:
		return left.ordinal < right.ordinal
	case left.predeclared != right.predeclared:
		return left.predeclared < right.predeclared
	case left.owner != right.owner:
		return left.owner < right.owner
	default:
		return left.occurrence < right.occurrence
	}
}

func lessStoredBinding(
	left storedBindingIdentity,
	right storedBindingIdentity,
) bool {
	switch {
	case left.owner != right.owner:
		return left.owner < right.owner
	case left.declaration != right.declaration:
		return left.declaration < right.declaration
	case left.role != right.role:
		return left.role < right.role
	default:
		return left.ordinal < right.ordinal
	}
}

func lessStoredOperation(
	left storedOperationIdentity,
	right storedOperationIdentity,
) bool {
	switch {
	case left.definition != right.definition:
		return left.definition < right.definition
	case left.occurrence != right.occurrence:
		return left.occurrence < right.occurrence
	case left.implicit != right.implicit:
		return left.implicit < right.implicit
	default:
		return left.ordinal < right.ordinal
	}
}
