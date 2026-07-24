package semantic

import "fmt"

type binaryIdentityReference interface {
	~uint64
	markBinaryUse(*binaryIdentityUses)
}

type binaryIdentityUses struct {
	table        packageIdentityTable
	modules      []bool
	owners       []bool
	packages     []bool
	files        []bool
	spans        []bool
	occurrences  []bool
	definitions  []bool
	types        []bool
	declarations []bool
	bindings     []bool
	operations   []bool
	unsupported  []bool
}

func newBinaryIdentityUses(table packageIdentityTable) *binaryIdentityUses {
	return &binaryIdentityUses{
		table:        table,
		modules:      make([]bool, len(table.modules)),
		owners:       make([]bool, len(table.owners)),
		packages:     make([]bool, len(table.packages)),
		files:        make([]bool, len(table.files)),
		spans:        make([]bool, len(table.spans)),
		occurrences:  make([]bool, len(table.occurrences)),
		definitions:  make([]bool, len(table.definitions)),
		types:        make([]bool, len(table.types)),
		declarations: make([]bool, len(table.declarations)),
		bindings:     make([]bool, len(table.bindings)),
		operations:   make([]bool, len(table.operations)),
		unsupported:  make([]bool, len(table.unsupported)),
	}
}

func readIdentityReference[Reference binaryIdentityReference](
	decoder *binaryShardDecoder,
	name string,
) (Reference, error) {
	value, err := readUnsignedAs[Reference](decoder, name)
	if err == nil {
		value.markBinaryUse(decoder.identityUses)
	}
	return value, err
}

func (reference moduleRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil || !markBinaryReference(uses.modules, reference) {
		return
	}
}

func (reference ownerRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil || !markBinaryReference(uses.owners, reference) {
		return
	}
	record := uses.table.owners[reference-1]
	record.module.markBinaryUse(uses)
}

func (reference packageRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil || !markBinaryReference(uses.packages, reference) {
		return
	}
	record := uses.table.packages[reference-1]
	record.owner.markBinaryUse(uses)
}

func (reference fileRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil || !markBinaryReference(uses.files, reference) {
		return
	}
	record := uses.table.files[reference-1]
	record.owner.markBinaryUse(uses)
}

func (reference spanRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil || !markBinaryReference(uses.spans, reference) {
		return
	}
	record := uses.table.spans[reference-1]
	record.file.markBinaryUse(uses)
}

func (reference occurrenceRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil ||
		!markBinaryReference(uses.occurrences, reference) {
		return
	}
	record := uses.table.occurrences[reference-1]
	record.span.markBinaryUse(uses)
}

func (reference definitionRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil ||
		!markBinaryReference(uses.definitions, reference) {
		return
	}
	record := uses.table.definitions[reference-1]
	record.root.markBinaryUse(uses)
	record.pkg.markBinaryUse(uses)
}

func (reference typeRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil {
		return
	}
	markBinaryReference(uses.types, reference)
}

func (reference declarationRef) markBinaryUse(
	uses *binaryIdentityUses,
) {
	if uses == nil ||
		!markBinaryReference(uses.declarations, reference) {
		return
	}
	record := uses.table.declarations[reference-1]
	record.pkg.markBinaryUse(uses)
	record.ownerType.markBinaryUse(uses)
	record.memberPkg.markBinaryUse(uses)
	record.owner.markBinaryUse(uses)
	record.occurrence.markBinaryUse(uses)
}

func (reference bindingRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil || !markBinaryReference(uses.bindings, reference) {
		return
	}
	record := uses.table.bindings[reference-1]
	record.owner.markBinaryUse(uses)
	record.declaration.markBinaryUse(uses)
}

func (reference operationRef) markBinaryUse(uses *binaryIdentityUses) {
	if uses == nil ||
		!markBinaryReference(uses.operations, reference) {
		return
	}
	record := uses.table.operations[reference-1]
	record.definition.markBinaryUse(uses)
	record.occurrence.markBinaryUse(uses)
}

func (reference unsupportedRef) markBinaryUse(
	uses *binaryIdentityUses,
) {
	if uses == nil ||
		!markBinaryReference(uses.unsupported, reference) {
		return
	}
	record := uses.table.unsupported[reference-1]
	record.definition.markBinaryUse(uses)
	record.occurrence.markBinaryUse(uses)
}

func markBinaryReference[Reference ~uint64](
	used []bool,
	reference Reference,
) bool {
	if reference == 0 || uint64(reference) > uint64(len(used)) {
		return false
	}
	index := uint64(reference) - 1
	if used[index] {
		return false
	}
	used[index] = true
	return true
}

func (uses *binaryIdentityUses) complete() error {
	if uses == nil {
		return fmt.Errorf("semantic binary identity usage is absent")
	}
	tables := []struct {
		name string
		used []bool
	}{
		{name: "module", used: uses.modules},
		{name: "owner", used: uses.owners},
		{name: "package", used: uses.packages},
		{name: "file", used: uses.files},
		{name: "span", used: uses.spans},
		{name: "occurrence", used: uses.occurrences},
		{name: "definition", used: uses.definitions},
		{name: "type", used: uses.types},
		{name: "declaration", used: uses.declarations},
		{name: "binding", used: uses.bindings},
		{name: "operation", used: uses.operations},
		{name: "unsupported", used: uses.unsupported},
	}
	for _, table := range tables {
		for index, used := range table.used {
			if !used {
				return fmt.Errorf(
					"semantic binary %s identity %d is unreferenced",
					table.name,
					index+1,
				)
			}
		}
	}
	return nil
}
