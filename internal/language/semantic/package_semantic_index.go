package semantic

import "fmt"

type normalizedPackageIndex struct {
	definitions  []bool
	resolutions  []bool
	declarations []bool
	bindings     []bool
	types        []bool
	witnesses    []bool
	operations   []bool
	unsupported  []bool
}

func newNormalizedPackageIndex(
	pkg Package,
) (normalizedPackageIndex, error) {
	definitions, err := canonicalRecordSet(
		"definition",
		len(pkg.identities.definitions),
		pkg.definitions.records,
		func(record storedDefinition) definitionRef {
			return record.id
		},
		func(reference definitionRef) string {
			return pkg.identities.definition(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	resolutions, err := canonicalRecordSet(
		"resolution",
		len(pkg.identities.occurrences),
		pkg.resolutions.records,
		func(record storedResolution) occurrenceRef {
			return record.occurrence
		},
		func(reference occurrenceRef) string {
			return pkg.identities.occurrence(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	declarations, err := canonicalRecordSet(
		"declaration",
		len(pkg.identities.declarations),
		pkg.declarations.records,
		func(record storedDeclaration) declarationRef {
			return record.id
		},
		func(reference declarationRef) string {
			return pkg.identities.declaration(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	bindings, err := canonicalRecordSet(
		"binding",
		len(pkg.identities.bindings),
		pkg.bindings.records,
		func(record storedBinding) bindingRef {
			return record.id
		},
		func(reference bindingRef) string {
			return pkg.identities.binding(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	types, err := canonicalRecordSet(
		"type",
		len(pkg.identities.types),
		pkg.types.records,
		func(record storedType) typeRef {
			return record.id
		},
		func(reference typeRef) string {
			return pkg.identities.typeID(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	witnesses, err := canonicalRecordSet(
		"type-witness",
		len(pkg.identities.types),
		pkg.witnesses.records,
		func(record storedTypeWitness) typeRef {
			return record.typeID
		},
		func(reference typeRef) string {
			return pkg.identities.typeID(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	operations, err := canonicalRecordSet(
		"operation",
		len(pkg.identities.operations),
		pkg.operations.records,
		func(record storedOperation) operationRef {
			return record.id
		},
		func(reference operationRef) string {
			return pkg.identities.operation(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	unsupported, err := canonicalRecordSet(
		"unsupported",
		len(pkg.identities.unsupported),
		pkg.unsupported.records,
		func(record storedUnsupported) unsupportedRef {
			return record.id
		},
		func(reference unsupportedRef) string {
			return pkg.identities.unsupportedID(reference).String()
		},
	)
	if err != nil {
		return normalizedPackageIndex{}, err
	}
	return normalizedPackageIndex{
		definitions: definitions, resolutions: resolutions,
		declarations: declarations, bindings: bindings,
		types: types, witnesses: witnesses,
		operations: operations, unsupported: unsupported,
	}, nil
}

func canonicalRecordSet[Reference ~uint64, Record any](
	name string,
	identityCount int,
	records []Record,
	reference func(Record) Reference,
	render func(Reference) string,
) ([]bool, error) {
	present := make([]bool, identityCount+1)
	var previous Reference
	for index, record := range records {
		current := reference(record)
		if current == 0 ||
			uint64(current) > uint64(identityCount) ||
			(index != 0 && current <= previous) {
			return nil, fmt.Errorf(
				"semantic %s records are not canonical at %d: current=%s (ref=%d), previous=%s (ref=%d)",
				name,
				index,
				render(current),
				current,
				render(previous),
				previous,
			)
		}
		present[current] = true
		previous = current
	}
	return present, nil
}

func referenceInSet[Reference ~uint64](
	present []bool,
	reference Reference,
) bool {
	return reference != 0 &&
		uint64(reference) < uint64(len(present)) &&
		present[reference]
}
